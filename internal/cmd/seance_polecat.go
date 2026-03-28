package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/checkpoint"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
)

// SeancePolecat represents a live polecat state report.
type SeancePolecat struct {
	Rig         string             `json:"rig"`
	Polecat     string             `json:"polecat"`
	PaneTail    []string           `json:"pane_tail"`
	GitState    *SeanceGitState    `json:"git_state"`
	FormulaStep *SeanceFormulaStep `json:"formula_step,omitempty"`
	Bead        *SeanceBeadStatus  `json:"bead,omitempty"`
	Session     *SeanceVitals      `json:"session"`
}

// SeanceGitState holds git state for a polecat's worktree.
type SeanceGitState struct {
	Branch           string   `json:"branch"`
	LastCommit       string   `json:"last_commit,omitempty"`
	UncommittedFiles []string `json:"uncommitted_files"`
	UnpushedCommits  int      `json:"unpushed_commits"`
}

// SeanceFormulaStep holds the current molecule formula step.
type SeanceFormulaStep struct {
	MoleculeID string `json:"molecule_id"`
	Formula    string `json:"formula,omitempty"`
	StepID     string `json:"step_id,omitempty"`
	StepTitle  string `json:"step_title,omitempty"`
}

// SeanceBeadStatus holds the assigned bead's title and status.
type SeanceBeadStatus struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// SeanceVitals holds session age and last-activity info.
type SeanceVitals struct {
	SessionID    string    `json:"session_id"`
	Running      bool      `json:"running"`
	AgeSeconds   int64     `json:"age_seconds,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`
}

// runSeancePolecat inspects a live polecat session without interrupting it.
func runSeancePolecat(rigName, polecatName string, lines int, paneOnly, jsonOut bool) error {
	report, err := gatherSeancePolecat(rigName, polecatName, lines)
	if err != nil {
		return err
	}

	if paneOnly {
		fmt.Print(strings.Join(report.PaneTail, "\n"))
		if len(report.PaneTail) > 0 {
			fmt.Println()
		}
		return nil
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	return printSeancePolecat(report)
}

// gatherSeancePolecat collects all state for a polecat seance report.
func gatherSeancePolecat(rigName, polecatName string, lines int) (*SeancePolecat, error) {
	mgr, r, err := getPolecatManager(rigName)
	if err != nil {
		return nil, err
	}

	p, err := mgr.Get(polecatName)
	if err != nil {
		return nil, fmt.Errorf("polecat '%s' not found in rig '%s'", polecatName, rigName)
	}

	t := tmux.NewTmux()
	sessManager := polecat.NewSessionManager(t, r)
	sessionName := sessManager.SessionName(polecatName)

	report := &SeancePolecat{
		Rig:     rigName,
		Polecat: polecatName,
	}

	// 1. Pane tail — last N lines of the tmux pane.
	if paneContent, captureErr := t.CapturePane(sessionName, lines); captureErr == nil {
		paneLines := strings.Split(paneContent, "\n")
		// Trim trailing empty lines for cleaner output.
		for len(paneLines) > 0 && paneLines[len(paneLines)-1] == "" {
			paneLines = paneLines[:len(paneLines)-1]
		}
		if len(paneLines) > lines {
			paneLines = paneLines[len(paneLines)-lines:]
		}
		report.PaneTail = paneLines
	} else {
		report.PaneTail = []string{"(session not running or pane unavailable)"}
	}

	// 2. Git state — branch, last commit, uncommitted changes, unpushed commits.
	gitState, _ := getGitState(p.ClonePath)
	if gitState != nil {
		state := &SeanceGitState{
			UncommittedFiles: gitState.UncommittedFiles,
			UnpushedCommits:  gitState.UnpushedCommits,
		}
		worktreeGit := git.NewGit(p.ClonePath)
		if branch, branchErr := worktreeGit.CurrentBranch(); branchErr == nil {
			state.Branch = branch
		}
		if lastCommit, commitErr := seanceLastCommitSubject(p.ClonePath); commitErr == nil {
			state.LastCommit = lastCommit
		}
		report.GitState = state
	}

	// 3. Formula step — from checkpoint file, falling back to bead description.
	report.FormulaStep = seanceReadFormulaStep(p, r)

	// 4. Bead status — assigned bead title + status.
	bd := beads.New(r.Path)
	agentBeadID := polecatBeadIDForRig(r, rigName, polecatName)
	agentIssue, agentFields, _ := bd.GetAgentBead(agentBeadID)

	hookBead := ""
	if agentIssue != nil {
		hookBead = agentIssue.HookBead
	}
	if hookBead == "" && agentFields != nil {
		hookBead = agentFields.HookBead
	}
	if hookBead != "" {
		if issue, showErr := bd.Show(hookBead); showErr == nil && issue != nil {
			report.Bead = &SeanceBeadStatus{
				ID:     issue.ID,
				Title:  issue.Title,
				Status: issue.Status,
			}
		}
	}

	// 5. Session vitals — session age and last activity.
	sessInfo, _ := sessManager.Status(polecatName)
	if sessInfo != nil {
		vitals := &SeanceVitals{
			SessionID:    sessInfo.SessionID,
			Running:      sessInfo.Running,
			LastActivity: sessInfo.LastActivity,
		}
		if !sessInfo.Created.IsZero() {
			vitals.AgeSeconds = int64(time.Since(sessInfo.Created).Seconds())
		}
		report.Session = vitals
	}

	return report, nil
}

// seanceLastCommitSubject returns the subject line of the last commit.
func seanceLastCommitSubject(worktreePath string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// seanceReadFormulaStep reads the current formula step, trying the checkpoint
// file first and falling back to the hook bead's attachment fields.
func seanceReadFormulaStep(p *polecat.Polecat, r *rig.Rig) *SeanceFormulaStep {
	// Try checkpoint in the polecat's git worktree (normal working directory).
	if cp, err := checkpoint.Read(p.ClonePath); err == nil && cp != nil && cp.MoleculeID != "" {
		return &SeanceFormulaStep{
			MoleculeID: cp.MoleculeID,
			StepID:     cp.CurrentStep,
			StepTitle:  cp.StepTitle,
		}
	}

	// Try checkpoint in polecatDir (fallback for older installs where cwd is parent).
	polecatDir := filepath.Join(r.Path, "polecats", p.Name)
	if cp, err := checkpoint.Read(polecatDir); err == nil && cp != nil && cp.MoleculeID != "" {
		return &SeanceFormulaStep{
			MoleculeID: cp.MoleculeID,
			StepID:     cp.CurrentStep,
			StepTitle:  cp.StepTitle,
		}
	}

	// Fall back to reading the attached_molecule from the hook bead's description.
	bd := beads.New(r.Path)
	agentBeadID := polecatBeadIDForRig(r, r.Name, p.Name)
	agentIssue, agentFields, _ := bd.GetAgentBead(agentBeadID)

	hookBead := ""
	if agentIssue != nil {
		hookBead = agentIssue.HookBead
	}
	if hookBead == "" && agentFields != nil {
		hookBead = agentFields.HookBead
	}
	if hookBead == "" {
		return nil
	}
	hookIssue, err := bd.Show(hookBead)
	if err != nil || hookIssue == nil {
		return nil
	}
	attachment := beads.ParseAttachmentFields(hookIssue)
	if attachment == nil || attachment.AttachedMolecule == "" {
		return nil
	}
	return &SeanceFormulaStep{
		MoleculeID: attachment.AttachedMolecule,
		Formula:    attachment.AttachedFormula,
	}
}

// printSeancePolecat renders the seance report in human-readable form.
func printSeancePolecat(r *SeancePolecat) error {
	const divider = "────────────────────────────────────────"

	// Header
	fmt.Printf("%s\n", style.Bold.Render(fmt.Sprintf("Seance: %s/%s", r.Rig, r.Polecat)))
	fmt.Printf("%s\n\n", divider)

	// Pane tail
	fmt.Printf("%s\n", style.Bold.Render("Pane (last output)"))
	fmt.Println(divider)
	fmt.Println(strings.Join(r.PaneTail, "\n"))
	fmt.Printf("%s\n\n", divider)

	// Git state
	fmt.Printf("%s\n", style.Bold.Render("Git State"))
	if r.GitState != nil {
		fmt.Printf("  Branch:      %s\n", style.Info.Render(r.GitState.Branch))
		if r.GitState.LastCommit != "" {
			fmt.Printf("  Last commit: %s\n", style.Dim.Render(r.GitState.LastCommit))
		}
		if r.GitState.UnpushedCommits > 0 {
			fmt.Printf("  Unpushed:    %s\n", style.Warning.Render(fmt.Sprintf("%d commit(s)", r.GitState.UnpushedCommits)))
		} else {
			fmt.Printf("  Unpushed:    %s\n", style.Dim.Render("0"))
		}
		if len(r.GitState.UncommittedFiles) > 0 {
			fmt.Printf("  Uncommitted: %s\n", style.Warning.Render(fmt.Sprintf("%d file(s)", len(r.GitState.UncommittedFiles))))
		} else {
			fmt.Printf("  Uncommitted: %s\n", style.Success.Render("clean"))
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(unavailable)"))
	}

	// Formula step
	fmt.Printf("\n%s\n", style.Bold.Render("Formula Step"))
	if r.FormulaStep != nil {
		fmt.Printf("  Molecule:    %s\n", r.FormulaStep.MoleculeID)
		if r.FormulaStep.Formula != "" {
			fmt.Printf("  Formula:     %s\n", r.FormulaStep.Formula)
		}
		if r.FormulaStep.StepID != "" {
			stepStr := r.FormulaStep.StepID
			if r.FormulaStep.StepTitle != "" {
				stepStr = fmt.Sprintf("%s (%s)", r.FormulaStep.StepID, r.FormulaStep.StepTitle)
			}
			fmt.Printf("  Step:        %s\n", stepStr)
		} else {
			fmt.Printf("  Step:        %s\n", style.Dim.Render("(no checkpoint)"))
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(no molecule attached)"))
	}

	// Bead status
	fmt.Printf("\n%s\n", style.Bold.Render("Bead Status"))
	if r.Bead != nil {
		statusStr := r.Bead.Status
		switch r.Bead.Status {
		case "in_progress":
			statusStr = style.Info.Render(statusStr)
		case "closed":
			statusStr = style.Success.Render(statusStr)
		case "blocked":
			statusStr = style.Warning.Render(statusStr)
		case "hooked":
			statusStr = style.Info.Render(statusStr)
		}
		fmt.Printf("  %s: %s [%s]\n", r.Bead.ID, r.Bead.Title, statusStr)
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(no bead assigned)"))
	}

	// Session vitals
	fmt.Printf("\n%s\n", style.Bold.Render("Session Vitals"))
	if r.Session != nil {
		if r.Session.Running {
			fmt.Printf("  Session ID:    %s\n", style.Dim.Render(r.Session.SessionID))
			if r.Session.AgeSeconds > 0 {
				fmt.Printf("  Age:           %s\n", seanceFormatDuration(time.Duration(r.Session.AgeSeconds)*time.Second))
			}
			if !r.Session.LastActivity.IsZero() {
				ago := formatActivityTime(r.Session.LastActivity)
				fmt.Printf("  Last activity: %s (%s)\n",
					r.Session.LastActivity.Format("15:04:05"),
					style.Dim.Render(ago))
			}
		} else {
			fmt.Printf("  Status: %s\n", style.Dim.Render("not running"))
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(unavailable)"))
	}

	return nil
}

// seanceFormatDuration formats a duration in human-readable form.
func seanceFormatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
