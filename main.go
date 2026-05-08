package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"timetracker/color"
	"timetracker/store"
	"timetracker/util"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "start":
		err = cmdStart(args)
	case "stop":
		err = cmdStop(args)
	case "status", "st":
		err = cmdStatus(args)
	case "log", "ls":
		err = cmdLog(args)
	case "report", "rep":
		err = cmdReport(args)
	case "delete", "del", "rm":
		err = cmdDelete(args)
	case "edit":
		err = cmdEdit(args)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "  Unknown command: %q\n  Run 'track help' for usage.\n\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "\n  "+color.Red("Error: ")+err.Error()+"\n")
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────────────────────────
// HELP
// ─────────────────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(`
  ` + color.BoldWhite("⏱  track") + `  — CLI Time Tracker

  ` + color.Yellow("Usage:") + `
    track <command> [options]

  ` + color.Yellow("Commands:") + `
    ` + color.Cyan("start") + ` <task> [-p project]   Start a new session
    ` + color.Cyan("stop") + `                         Stop the current session
    ` + color.Cyan("status") + `                       Show running session
    ` + color.Cyan("log") + ` [-d date]                Show sessions for a day
    ` + color.Cyan("report") + ` [--week|--month]       Summary report
    ` + color.Cyan("edit") + ` <id> [-t task] [-p proj] Edit a session
    ` + color.Cyan("delete") + ` <id>                   Delete a session

  ` + color.Yellow("Examples:") + `
    track start "Fix login bug" -p backend
    track stop
    track status
    track log
    track log -d yesterday
    track log -d 2026-05-01
    track report --week
    track report --month
    track edit a1b2c3d4 -t "New task name"
    track delete a1b2c3d4

  ` + color.Black("Data stored in: ~/.timetracker/data.json") + `
`)
}

// ─────────────────────────────────────────────────────────────────
// START
// ─────────────────────────────────────────────────────────────────

func cmdStart(args []string) error {
	// Extract -p / --project flag from anywhere in args
	var projectVal string
	var taskParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if (a == "-p" || a == "--project") && i+1 < len(args) {
			projectVal = args[i+1]
			i++
		} else if strings.HasPrefix(a, "-p=") {
			projectVal = strings.TrimPrefix(a, "-p=")
		} else if strings.HasPrefix(a, "--project=") {
			projectVal = strings.TrimPrefix(a, "--project=")
		} else {
			taskParts = append(taskParts, a)
		}
	}
	if len(taskParts) == 0 {
		return fmt.Errorf("usage: track start <task> [-p project]")
	}
	task := strings.Join(taskParts, " ")
	project := &projectVal

	s, err := store.New()
	if err != nil {
		return err
	}
	sess, err := s.Start(task, *project)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  %s  %s", color.BoldGreen("▶ Started"), color.BoldCyan(sess.Task))
	if sess.Project != "" {
		fmt.Printf("  %s", color.Yellow("["+sess.Project+"]"))
	}
	fmt.Println()
	fmt.Printf("  %s  %s\n", color.Black("  ID:"), color.White(sess.ID))
	fmt.Printf("  %s  %s\n", color.Black("  At:"), sess.StartTime.Format("15:04:05"))
	fmt.Println()
	return nil
}

// ─────────────────────────────────────────────────────────────────
// STOP
// ─────────────────────────────────────────────────────────────────

func cmdStop(args []string) error {
	s, err := store.New()
	if err != nil {
		return err
	}
	sess, err := s.Stop()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  %s  %s", color.BoldRed("■ Stopped"), color.BoldCyan(sess.Task))
	if sess.Project != "" {
		fmt.Printf("  %s", color.Yellow("["+sess.Project+"]"))
	}
	fmt.Println()
	fmt.Printf("  %s  %s → %s\n",
		color.Black("  Time:  "),
		sess.StartTime.Format("15:04"),
		sess.EndTime.Format("15:04"),
	)
	fmt.Printf("  %s  %s\n",
		color.Black("  Logged:"),
		color.BoldGreen(util.FormatDuration(sess.Duration())),
	)
	fmt.Println()
	return nil
}

// ─────────────────────────────────────────────────────────────────
// STATUS
// ─────────────────────────────────────────────────────────────────

func cmdStatus(_ []string) error {
	s, err := store.New()
	if err != nil {
		return err
	}
	sess := s.Active()
	if sess == nil {
		fmt.Println()
		fmt.Println("  " + color.Black("No active session.") + " Run 'track start <task>' to begin.")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println("  " + color.BoldGreen("● RUNNING"))
	fmt.Printf("  %s  %s\n", color.Black("Task:   "), color.BoldCyan(sess.Task))
	if sess.Project != "" {
		fmt.Printf("  %s  %s\n", color.Black("Project:"), color.Yellow(sess.Project))
	}
	fmt.Printf("  %s  %s\n", color.Black("Started:"), sess.StartTime.Format("15:04:05"))
	fmt.Printf("  %s  %s\n", color.Black("Elapsed:"), color.BoldGreen(util.FormatDuration(sess.Duration())))
	fmt.Printf("  %s  %s\n", color.Black("ID:     "), color.Black(sess.ID))
	fmt.Println()
	return nil
}

// ─────────────────────────────────────────────────────────────────
// LOG
// ─────────────────────────────────────────────────────────────────

func cmdLog(args []string) error {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	dateFlag := fs.String("d", "today", "Date: today, yesterday, or YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var target time.Time
	var err error
	switch *dateFlag {
	case "today", "":
		target = time.Now()
	case "yesterday":
		target = time.Now().AddDate(0, 0, -1)
	default:
		target, err = time.Parse("2006-01-02", *dateFlag)
		if err != nil {
			return fmt.Errorf("invalid date %q — use YYYY-MM-DD, 'today', or 'yesterday'", *dateFlag)
		}
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	sessions := s.ForDate(target)

	fmt.Println()
	fmt.Println("  " + color.BoldWhite("📅  "+target.Format("Monday, January 2 2006")))
	fmt.Println()

	if len(sessions) == 0 {
		fmt.Println("  " + color.Black("No sessions recorded."))
		fmt.Println()
		return nil
	}

	sep := color.Black("  " + strings.Repeat("─", 65))
	fmt.Printf("  %s  %-9s  %-10s  %-28s  %s\n",
		color.Black("ID      "),
		color.Black("Start"),
		color.Black("Duration"),
		color.Black("Task"),
		color.Black("Project"),
	)
	fmt.Println(sep)

	var total time.Duration
	for _, sess := range sessions {
		dur := sess.Duration()
		total += dur

		endStr := "ongoing"
		if sess.EndTime != nil {
			endStr = sess.EndTime.Format("15:04")
		}
		timeRange := sess.StartTime.Format("15:04") + "–" + endStr

		durStr := util.FormatDurationShort(dur)
		if sess.IsActive() {
			durStr = color.BoldGreen(durStr + " ▶")
		} else {
			durStr = color.Cyan(durStr)
		}

		proj := color.Black("—")
		if sess.Project != "" {
			proj = color.Yellow(sess.Project)
		}

		fmt.Printf("  %s  %-9s  %-22s  %-28s  %s\n",
			color.Black(sess.ID),
			timeRange,
			durStr,
			util.Truncate(sess.Task, 28),
			proj,
		)
	}

	fmt.Println(sep)
	pluralS := ""
	if len(sessions) != 1 {
		pluralS = "s"
	}
	fmt.Printf("  %s  %s  %s\n\n",
		color.Black("Total:"),
		color.BoldGreen(util.FormatDurationShort(total)),
		color.Black(fmt.Sprintf("(%d session%s)", len(sessions), pluralS)),
	)
	return nil
}

// ─────────────────────────────────────────────────────────────────
// REPORT
// ─────────────────────────────────────────────────────────────────

func cmdReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	weekFlag := fs.Bool("week", false, "Weekly report")
	monthFlag := fs.Bool("month", false, "Monthly report")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}

	now := time.Now()
	var sessions []store.Session
	var title string

	if *monthFlag {
		sessions = s.ForMonth(now)
		title = "Monthly Report — " + now.Format("January 2006")
	} else {
		sessions = s.ForWeek(now)
		wd := int(now.Weekday())
		if wd == 0 {
			wd = 7
		}
		ws := now.AddDate(0, 0, -(wd-1))
		we := ws.AddDate(0, 0, 6)
		title = fmt.Sprintf("Weekly Report — %s to %s", ws.Format("Jan 2"), we.Format("Jan 2, 2006"))
		_ = weekFlag
	}

	fmt.Println()
	fmt.Println("  " + color.BoldWhite("📊  "+title))
	fmt.Println()

	if len(sessions) == 0 {
		fmt.Println("  " + color.Black("No sessions in this period."))
		fmt.Println()
		return nil
	}

	// ── By Day ───────────────────────────────────────────────────
	fmt.Println("  " + color.Yellow("By Day"))
	fmt.Println()

	dayMap := make(map[string]time.Duration)
	var dayOrder []string
	for _, sess := range sessions {
		key := sess.StartTime.Format("2006-01-02")
		if _, ok := dayMap[key]; !ok {
			dayOrder = append(dayOrder, key)
		}
		dayMap[key] += sess.Duration()
	}
	sort.Strings(dayOrder)

	var maxDay time.Duration
	var grandTotal time.Duration
	for _, d := range dayMap {
		if d > maxDay {
			maxDay = d
		}
		grandTotal += d
	}

	for _, day := range dayOrder {
		dur := dayMap[day]
		t, _ := time.Parse("2006-01-02", day)
		fraction := float64(dur) / float64(maxDay)
		bar := util.Bar(fraction, 20)
		fmt.Printf("  %-12s  %s  %s  %s\n",
			color.White(t.Format("Mon Jan 02")),
			color.Cyan(bar),
			color.BoldGreen(fmt.Sprintf("%-8s", util.FormatDurationShort(dur))),
			color.Black(util.FormatDurationHours(dur)),
		)
	}
	fmt.Println()

	// ── By Project ───────────────────────────────────────────────
	projMap := make(map[string]time.Duration)
	for _, sess := range sessions {
		key := sess.Project
		if key == "" {
			key = "(no project)"
		}
		projMap[key] += sess.Duration()
	}

	if len(projMap) > 0 {
		type pe struct {
			name string
			dur  time.Duration
		}
		var entries []pe
		for k, v := range projMap {
			entries = append(entries, pe{k, v})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].dur > entries[j].dur })

		fmt.Println("  " + color.Yellow("By Project"))
		fmt.Println()
		for _, e := range entries {
			pct := float64(e.dur) / float64(grandTotal)
			bar := util.Bar(pct, 20)
			name := color.Yellow(e.name)
			if e.name == "(no project)" {
				name = color.Black(e.name)
			}
			fmt.Printf("  %-30s  %s  %-10s  %s\n",
				name,
				color.Cyan(bar),
				color.BoldGreen(util.FormatDurationShort(e.dur)),
				color.Black(fmt.Sprintf("%.0f%%", pct*100)),
			)
		}
		fmt.Println()
	}

	// ── Summary ──────────────────────────────────────────────────
	fmt.Println("  " + color.Black(strings.Repeat("─", 45)))
	pluralS := ""
	if len(sessions) != 1 {
		pluralS = "s"
	}
	fmt.Printf("  %s  %s  %s\n",
		color.Black("Total:"),
		color.BoldGreen(util.FormatDurationShort(grandTotal)),
		color.Black(fmt.Sprintf("across %d session%s", len(sessions), pluralS)),
	)
	if n := len(dayOrder); n > 0 {
		avg := grandTotal / time.Duration(n)
		fmt.Printf("  %s  %s %s\n",
			color.Black("Avg:  "),
			color.Cyan(util.FormatDurationShort(avg)),
			color.Black("/ day"),
		)
	}
	fmt.Println()
	return nil
}

// ─────────────────────────────────────────────────────────────────
// DELETE
// ─────────────────────────────────────────────────────────────────

func cmdDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: track delete <id>")
	}
	id := args[0]
	s, err := store.New()
	if err != nil {
		return err
	}
	if err := s.Delete(id); err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("  %s  session %s deleted\n\n", color.Red("✗"), color.White(id))
	return nil
}

// ─────────────────────────────────────────────────────────────────
// EDIT
// ─────────────────────────────────────────────────────────────────

func cmdEdit(args []string) error {
	var id, taskVal, projVal string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case (a == "-t" || a == "--task") && i+1 < len(args):
			taskVal = args[i+1]; i++
		case strings.HasPrefix(a, "-t="):
			taskVal = strings.TrimPrefix(a, "-t=")
		case (a == "-p" || a == "--project") && i+1 < len(args):
			projVal = args[i+1]; i++
		case strings.HasPrefix(a, "-p="):
			projVal = strings.TrimPrefix(a, "-p=")
		default:
			if id == "" {
				id = a
			}
		}
	}
	if id == "" {
		return fmt.Errorf("usage: track edit <id> [-t task] [-p project]")
	}
	if taskVal == "" && projVal == "" {
		return fmt.Errorf("provide -t and/or -p to update")
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	sess, err := s.Edit(id, taskVal, projVal)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("  %s  Updated %s\n", color.BoldGreen("✓"), color.White(sess.ID))
	fmt.Printf("     Task:    %s\n", color.BoldCyan(sess.Task))
	if sess.Project != "" {
		fmt.Printf("     Project: %s\n", color.Yellow(sess.Project))
	}
	fmt.Println()
	return nil
}
