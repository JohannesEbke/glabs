package gitlab

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/obcode/glabs/config"
	"github.com/rs/zerolog/log"
	"github.com/theckman/yacspin"
	"github.com/xanzy/go-gitlab"
)

func getResultsJsPath(assignmentCfg *config.AssignmentConfig) string {
	return "results_" + string(regexp.MustCompile("\\/").ReplaceAll([]byte(assignmentCfg.Path), []byte{'_'})) + ".js"
}

func (c *Client) FetchResults(assignmentCfg *config.AssignmentConfig) {
	assignmentGitLabGroupID, err := c.getGroupID(assignmentCfg)
	if err != nil {
		fmt.Printf("error: GitLab group for assignment does not exist, please create the group %s\n", assignmentCfg.URL)
		os.Exit(1)
	}

	f, err := os.OpenFile(getResultsJsPath(assignmentCfg), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("cannot open output file")
	}
	f.WriteString("scoreMap = {};\n")
	f.Close()

	switch per := assignmentCfg.Per; per {
	case config.PerGroup:
		c.fetchResultsPerGroup(assignmentCfg, assignmentGitLabGroupID)
	case config.PerStudent:
		c.fetchResultsPerStudent(assignmentCfg, assignmentGitLabGroupID)
	default:
		fmt.Printf("it is only possible to generate for students oder groups, not for %v", per)
		os.Exit(1)
	}

	f, err = os.OpenFile(getResultsJsPath(assignmentCfg), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("cannot open output file")
	}

	f.WriteString("as = document.getElementsByTagName(\"a\");\n")
	f.WriteString("for (i = 0; i < as.length; i++) {\n")
	f.WriteString("    var score = scoreMap[as.item(i).innerText];\n")
	f.WriteString("    if (score != undefined) {\n")
	f.WriteString("        console.log(\"Setting score of \" + as.item(i).innerText + \" to \" + score);\n")
	f.WriteString("        as.item(i).parentElement.parentElement.childNodes[4].getElementsByTagName(\"input\")[0].value = score\n")
	f.WriteString("        delete scoreMap[as.item(i).innerText];\n")
	f.WriteString("    }\n")
	f.WriteString("}\n")
	f.WriteString("console.log(\"Scores which were not set:\")\n")
	f.WriteString("console.log(scoreMap)\n")
	f.Close()
}

func startSpinner(spinner *yacspin.Spinner) {
	err := spinner.Start()
	if err != nil {
		log.Debug().Err(err).Msg("cannot start spinner")
	}
}

func stopSpinner(spinner *yacspin.Spinner, err error) error {
	if err != nil {
		log.Debug().Err(err).Msg("Problem:")
		spinner.StopFailMessage(fmt.Sprintf("problem: %v", err))

		err := spinner.StopFail()
		if err != nil {
			log.Debug().Err(err).Msg("cannot stop spinner")
		}
	} else {
		err = spinner.Stop()
		if err != nil {
			log.Debug().Err(err).Msg("cannot stop spinner")
		}
	}
	return err
}

func (c *Client) fetchResults(assignmentCfg *config.AssignmentConfig, assignmentGroupID int,
	projectname string, members []string) {

	cfg := yacspin.Config{
		Frequency: 100 * time.Millisecond,
		CharSet:   yacspin.CharSets[69],
		Suffix: aurora.Sprintf(aurora.Cyan(" fetching project %s at %s"),
			aurora.Yellow(projectname),
			aurora.Magenta(assignmentCfg.URL+"/"+projectname),
		),
		SuffixAutoColon:   true,
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopFailMessage:   "error",
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
	}

	f, err := os.OpenFile(getResultsJsPath(assignmentCfg), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("cannot open output file")
	}
	defer f.Close()

	spinner, err := yacspin.New(cfg)
	if err != nil {
		log.Debug().Err(err).Msg("cannot create spinner")
	}
	startSpinner(spinner)
	spinner.Message("fetching results")

	fullprojectname := assignmentCfg.Path + "/" + projectname
	log.Debug().Err(err).Str("name", projectname).Msg("searching for project")
	project, err := c.getProjectByName(fullprojectname)
	if err != nil {
		stopSpinner(spinner, err)
		return
	}

	spinner.Message("fetching pipelines")
	pipeCfg := gitlab.ListProjectPipelinesOptions{
		Scope: gitlab.String("finished"),
	}
	pipelines, _, err := c.Pipelines.ListProjectPipelines(project.ID, &pipeCfg, nil)
	if err != nil {
		stopSpinner(spinner, err)
		return
	}
	if len(pipelines) == 0 {
		log.Info().Str("projectname", projectname).Msg("has no pipelines")
		return
	}
	pipeline := pipelines[0]
	jobs, _, err := c.Jobs.ListPipelineJobs(project.ID, pipeline.ID, nil)
	if err != nil {
		stopSpinner(spinner, err)
		return
	}
	for _, job := range jobs {
		if job.Name == "Summary Report" {
			reader, _, err := c.Jobs.DownloadSingleArtifactsFile(project.ID, job.ID, "grading.html", nil)
			if err != nil {
				stopSpinner(spinner, err)
				return
			}
			gradingHtml, err := ioutil.ReadAll(reader)
			if err != nil {
				stopSpinner(spinner, err)
				return
			}
			successfulTests, totalTests, score, err := parseGradingHtml(gradingHtml)
			if err != nil {
				stopSpinner(spinner, err)
				return
			}
			for _, student := range members {
				user, err := c.getUser(student)
				if err != nil {
					stopSpinner(spinner, err)
					return
				}
				log.Info().Int("s", successfulTests).Int("t", totalTests).Int("score", score).Msg("results")
				fmt.Fprintf(f, "// %v;%v;%v;%v;%v\n", user.Username, user.Name, successfulTests, totalTests, score)
				finalScore := score
				if 10*successfulTests < 9*totalTests {
					finalScore = 0
				}
				fmt.Fprintf(f, "scoreMap[\"%v\"] = %v;\n", user.Name, finalScore)
			}
			break
		}
	}
	stopSpinner(spinner, nil)
}

func (c *Client) fetchResultsPerStudent(assignmentCfg *config.AssignmentConfig, assignmentGroupID int) {
	if len(assignmentCfg.Students) == 0 {
		fmt.Println("no students in config for assignment found")
		return
	}

	for _, student := range assignmentCfg.Students {
		c.fetchResults(assignmentCfg, assignmentGroupID, assignmentCfg.Name+"-"+student, []string{student})
	}
}

func (c *Client) fetchResultsPerGroup(assignmentCfg *config.AssignmentConfig, assignmentGroupID int) {
	if len(assignmentCfg.Groups) == 0 {
		log.Info().Str("group", assignmentCfg.Course).Msg("no groups found")
		return
	}

	for _, grp := range assignmentCfg.Groups {
		c.fetchResults(assignmentCfg, assignmentGroupID, assignmentCfg.Name+"-"+grp.Name, grp.Members)
	}
}

func parseGradingHtml(gradingHtml []byte) (int, int, int, error) {
	testRegex := regexp.MustCompile(" \\(([0-9]+)\\/([0-9]+)\\)\\<\\/strong\\>")
	scoreRegex := regexp.MustCompile(" \\(([0-9]+)\\)\\<\\/strong\\>")

	testMatches := testRegex.FindSubmatch(gradingHtml)
	if len(testMatches) == 0 {
		return 0, 0, 0, fmt.Errorf("No matches for number of tests in grading.html")
	}
	scoreMatches := scoreRegex.FindSubmatch(gradingHtml)
	if len(scoreMatches) == 0 {
		return 0, 0, 0, fmt.Errorf("No matches for score of tests in grading.html")
	}

	successfulTests, err := strconv.Atoi(string(testMatches[1]))
	if err != nil {
		return 0, 0, 0, err
	}
	totalTests, err := strconv.Atoi(string(testMatches[2]))
	if err != nil {
		return 0, 0, 0, err
	}
	score, err := strconv.Atoi(string(scoreMatches[1]))
	if err != nil {
		return 0, 0, 0, err
	}
	return successfulTests, totalTests, score, nil
}
