package config

import (
	"fmt"
	"strings"

	"github.com/logrusorgru/aurora/v3"
)

func (cfg *AssignmentConfig) Show() {
	containerRegistry := aurora.Red("disabled")
	if cfg.ContainerRegistry {
		containerRegistry = aurora.Green("enabled")
	}

	startercode := aurora.Sprintf(aurora.Red("        not defined"))
	if cfg.Startercode != nil {
		startercode = aurora.Sprintf(aurora.Cyan(`
  URL:              %s
  FromBranch:       %s
  ToBranch:         %s
  ProtectToBranch:  %t`),
			aurora.Yellow(cfg.Startercode.URL),
			aurora.Yellow(cfg.Startercode.FromBranch),
			aurora.Yellow(cfg.Startercode.ToBranch),
			aurora.Yellow(cfg.Startercode.ProtectToBranch),
		)
	}

	clone := aurora.Sprintf(aurora.Red("        not defined"))
	if cfg.Clone != nil {
		clone = aurora.Sprintf(aurora.Cyan(`Clone:
  Localpath:        %s
  Branch:           %s`),
			aurora.Yellow(cfg.Clone.LocalPath),
			aurora.Yellow(cfg.Clone.Branch),
		)
	}

	var per strings.Builder
	switch cfg.Per {
	case PerStudent:
		per.WriteString(aurora.Sprintf(aurora.Cyan("Students:\n")))
		for _, s := range cfg.Students {
			per.WriteString(aurora.Sprintf(aurora.Cyan("  - ")))
			per.WriteString(aurora.Sprintf(aurora.Yellow(s)))
			per.WriteString("\n")
		}
	case PerGroup:
		per.WriteString(aurora.Sprintf(aurora.Cyan("Groups:\n")))
		for _, grp := range cfg.Groups {
			per.WriteString(aurora.Sprintf(aurora.Cyan("  - ")))
			per.WriteString(aurora.Sprintf(aurora.Yellow(grp.Name)))
			per.WriteString(aurora.Sprintf(aurora.Cyan(": ")))
			for i, m := range grp.Members {
				per.WriteString(aurora.Sprintf(aurora.Green(m)))
				if i == len(grp.Members)-1 {
					per.WriteString("\n")
				} else {
					per.WriteString(aurora.Sprintf(aurora.Cyan(", ")))
				}
			}
		}
	}

	groupsOrStudents := per.String()

	fmt.Print(aurora.Sprintf(aurora.Cyan(`
Course:             %s
Assignment:         %s
Per:                %s
Base-URL:           %s
Description:	    %s
AccessLevel:        %s
Container-Registry: %s
Startercode:%s
%s
%s
`),
		aurora.Yellow(cfg.Course),
		aurora.Yellow(cfg.Name),
		aurora.Yellow(cfg.Per),
		aurora.Yellow(cfg.URL),
		aurora.Yellow(cfg.Description),
		aurora.Yellow(cfg.AccessLevel.String()),
		containerRegistry,
		startercode,
		clone,
		groupsOrStudents,
	))

}
