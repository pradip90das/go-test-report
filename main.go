package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

//go:embed test_report.html.template
var testReportHTMLTemplateStr []byte

//go:embed test_report.js
var testReportJsCodeStr []byte

type Status struct {
	Pass        int    `json:"pass,omitempty"`
	Fail        int    `json:"fail,omitempty"`
	Total       int    `json:"total,omitempty"`
	Skip        int    `json:"skip,omitempty"`
	ElapsedTime string `json:"elapsed_time,omitempty"`
}

type (
	goTestOutputRow struct {
		Time        string
		TestName    string `json:"Test"`
		Action      string
		Package     string
		Elapsed     float64
		Output      string
		Screenshots []string
	}

	testStatus struct {
		TestName           string
		Package            string
		ElapsedTime        float64
		Output             []string
		Passed             bool
		Skipped            bool
		TestFileName       string
		TestFunctionDetail testFunctionFilePos
		Screenshots        []string
	}
	Info struct {
		Key, Value string
		IsLink     bool
	}

	templateData struct {
		TestResultGroupIndicatorWidth  string
		TestResultGroupIndicatorHeight string
		TestResults                    []*testGroupData
		NumOfTestPassed                int
		NumOfTestFailed                int
		NumOfTestSkipped               int
		NumOfTests                     int
		TestDuration                   time.Duration
		ReportTitle                    string
		JsCode                         template.JS
		numOfTestsPerGroup             int
		OutputFilename                 string
		InputFilename                  string
		TestExecutionDate              string
		ServerInfo                     []Info
	}

	testGroupData struct {
		FailureIndicator string
		SkippedIndicator string
		TestResults      []*testStatus
	}

	cmdFlags struct {
		serverInfo string
		titleFlag  string
		sizeFlag   string
		groupSize  int
		listFlag   string
		inputFlag  string
		outputFlag string
		outputEnv  string
		verbose    bool
	}

	goListJSONModule struct {
		Path string
		Dir  string
		Main bool
	}

	goListJSON struct {
		Dir         string
		ImportPath  string
		Name        string
		GoFiles     []string
		TestGoFiles []string
		Module      goListJSONModule
	}

	testFunctionFilePos struct {
		Line int
		Col  int
	}

	testFileDetail struct {
		FileName            string
		TestFunctionFilePos testFunctionFilePos
	}

	testFileDetailsByTest    map[string]*testFileDetail
	testFileDetailsByPackage map[string]testFileDetailsByTest
)

func main() {
	rootCmd, _, _ := initRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initRootCommand() (*cobra.Command, *templateData, *cmdFlags) {
	flags := &cmdFlags{}
	tmplData := &templateData{}
	rootCmd := &cobra.Command{
		Use:  "report",
		Long: "convert json go test report to html",
		RunE: func(cmd *cobra.Command, args []string) (e error) {

			startTime := time.Now()
			if err := parseSizeFlag(tmplData, flags); err != nil {
				return err
			}
			serverInfo := []Info{}
			for _, info := range strings.Split(flags.serverInfo, ";") {
				kv := strings.Split(info, "::")
				if len(kv) > 1 {
					var isLink bool
					if strings.Contains(kv[1], "http") {
						isLink = true
					}
					info := Info{Key: kv[0], Value: kv[1], IsLink: isLink}
					serverInfo = append(serverInfo, info)

				}
			}
			tmplData.ServerInfo = serverInfo

			tmplData.numOfTestsPerGroup = flags.groupSize
			tmplData.ReportTitle = flags.titleFlag
			tmplData.OutputFilename = flags.outputFlag
			tmplData.OutputFilename = flags.outputFlag
			tmplData.InputFilename = flags.inputFlag
			// if err := checkIfStdinIsPiped(); err != nil {
			// 	return err
			// }
			testReportHTMLTemplateFile, _ := os.Create(tmplData.OutputFilename)
			reportFileWriter := bufio.NewWriter(testReportHTMLTemplateFile)
			defer func() {
				if err := reportFileWriter.Flush(); err != nil {
					e = err
				}
				if err := testReportHTMLTemplateFile.Close(); err != nil {
					e = err
				}
			}()
			strStartTestTime := os.Getenv("START_TIME")
			startTestTime, err := time.Parse(time.UnixDate, strStartTestTime)
			if err != nil {
				startTestTime = time.Now()
			}
			allPackageNames, allTests, err := readTestDataFromFile(tmplData.InputFilename, flags, cmd)
			if err != nil {
				return errors.New("failed to read input file,err=" + err.Error() + "\n")
			}
			elapsedTestTime := time.Since(startTestTime)
			strElapsedTestTime := os.Getenv("END_TIME")
			endTestTime, err := time.Parse(time.UnixDate, strElapsedTestTime)
			if err == nil {
				elapsedTestTime = endTestTime.Sub(startTestTime)
			}
			// used to the location of test functions in test go files by package and test function name.
			var testFileDetailByPackage testFileDetailsByPackage
			if flags.listFlag != "" {

				testFileDetailByPackage, err = getAllDetails(flags.listFlag)
			} else {
				testFileDetailByPackage, err = getPackageDetails(allPackageNames)
			}
			if err != nil {
				return err
			}
			err = generateReport(tmplData, allTests, testFileDetailByPackage, elapsedTestTime, reportFileWriter, flags.outputEnv)
			if err != nil {
				return err
			}

			elapsedTime := time.Since(startTime)
			elapsedTimeMsg := []byte(fmt.Sprintf("[report] finished in %s\n", elapsedTime))
			if _, err := cmd.OutOrStdout().Write(elapsedTimeMsg); err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.PersistentFlags().StringVarP(&flags.serverInfo,
		"serverInfo",
		"f",
		"",
		"the server info shown in the test report")
	rootCmd.PersistentFlags().StringVarP(&flags.titleFlag,
		"title",
		"t",
		"report",
		"the title text shown in the test report")
	rootCmd.PersistentFlags().StringVarP(&flags.sizeFlag,
		"size",
		"s",
		"24",
		"the size (in pixels) of the clickable indicator for test result groups")
	rootCmd.PersistentFlags().IntVarP(&flags.groupSize,
		"groupSize",
		"g",
		20,
		"the number of tests per test group indicator")
	rootCmd.PersistentFlags().StringVarP(&flags.listFlag,
		"list",
		"l",
		"",
		"the JSON module list")
	rootCmd.PersistentFlags().StringVarP(&flags.outputFlag,
		"output",
		"o",
		"report.html",
		"the HTML output file")
	rootCmd.PersistentFlags().StringVarP(&flags.outputEnv,
		"env",
		"e",
		"status.env",
		"env file with total,pass,fail,skip status")
	rootCmd.PersistentFlags().StringVarP(&flags.inputFlag,
		"input",
		"i",
		"",
		"the json input file")
	rootCmd.MarkPersistentFlagRequired("input")
	rootCmd.PersistentFlags().BoolVarP(&flags.verbose,
		"verbose",
		"v",
		false,
		"while processing, show the complete output from go test ")

	return rootCmd, tmplData, flags
}

func readTestDataFromFile(inputFile string, flags *cmdFlags, cmd *cobra.Command) (allPackageNames map[string]*types.Nil, allTests map[string]*testStatus, e error) {
	allTests = map[string]*testStatus{}
	allPackageNames = map[string]*types.Nil{}
	testReportJsCodeByte, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("failed reading data from file: %s", err)
	}
	testReportJsCodeStr := string(testReportJsCodeByte)
	for _, lineInput := range strings.Split(testReportJsCodeStr, "\n") {
		if len(lineInput) < 1 {
			continue
		}
		goTestOutputRow := &goTestOutputRow{}
		if err := json.Unmarshal([]byte(lineInput), goTestOutputRow); err != nil {
			return nil, nil, err
		}
		if goTestOutputRow.TestName != "" {
			var status *testStatus
			key := goTestOutputRow.Package + "." + goTestOutputRow.TestName
			if _, exists := allTests[key]; !exists {
				status = &testStatus{
					TestName: goTestOutputRow.TestName,
					Package:  goTestOutputRow.Package,
					Output:   []string{},
				}
				allTests[key] = status
			} else {
				status = allTests[key]
			}
			if goTestOutputRow.Action == "pass" || goTestOutputRow.Action == "fail" || goTestOutputRow.Action == "skip" {
				if goTestOutputRow.Action == "pass" {
					status.Passed = true
				}
				if goTestOutputRow.Action == "skip" {
					status.Skipped = true
				}
				status.ElapsedTime = goTestOutputRow.Elapsed
			}
			allPackageNames[goTestOutputRow.Package] = nil
			if strings.Contains(goTestOutputRow.Output, "--- PASS:") {
				goTestOutputRow.Output = strings.TrimSpace(goTestOutputRow.Output)
			}
			screenshotKeyWord := "Screenshots :"
			if strings.Contains(goTestOutputRow.Output, screenshotKeyWord) {
				screenshots := strings.TrimSpace(strings.Split(goTestOutputRow.Output, screenshotKeyWord)[1])
				goTestOutputRow.Screenshots = append(goTestOutputRow.Screenshots, strings.Split(screenshots[1:len(screenshots)-1], " ")...)
			}
			status.Output = append(status.Output, goTestOutputRow.Output)
			status.Screenshots = append(status.Screenshots, goTestOutputRow.Screenshots...)
		}
	}
	return allPackageNames, allTests, nil
}

func getAllDetails(listFile string) (testFileDetailsByPackage, error) {
	testFileDetailByPackage := testFileDetailsByPackage{}
	f, err := os.Open(listFile)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	list := json.NewDecoder(f)
	for list.More() {
		goListJSON := goListJSON{}
		if err := list.Decode(&goListJSON); err != nil {
			return nil, err
		}
		packageName := goListJSON.ImportPath
		testFileDetailsByTest, err := getFileDetails(&goListJSON)
		if err != nil {
			return nil, err
		}
		testFileDetailByPackage[packageName] = testFileDetailsByTest
	}
	return testFileDetailByPackage, nil
}

func getPackageDetails(allPackageNames map[string]*types.Nil) (testFileDetailsByPackage, error) {
	var testFileDetailByPackage testFileDetailsByPackage
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	details := make(chan testFileDetailsByPackage)
	for packageName := range allPackageNames {
		name := packageName
		g.Go(func() error {
			testFileDetailsByTest, err := getTestDetails(name)
			if err != nil {
				return err
			}
			select {
			case details <- testFileDetailsByPackage{name: testFileDetailsByTest}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil

		})
	}
	go func() {
		g.Wait()
		close(details)
	}()

	testFileDetailByPackage = make(testFileDetailsByPackage, len(allPackageNames))
	for d := range details {
		for packageName, testFileDetailsByTest := range d {
			testFileDetailByPackage[packageName] = testFileDetailsByTest
		}
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return testFileDetailByPackage, nil
}

func getTestDetails(packageName string) (testFileDetailsByTest, error) {
	var out bytes.Buffer
	var cmd *exec.Cmd
	stringReader := strings.NewReader("")
	cmd = exec.Command("go", "list", "-json", packageName)
	cmd.Stdin = stringReader
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	goListJSON := &goListJSON{}
	if err := json.Unmarshal(out.Bytes(), goListJSON); err != nil {
		return nil, err
	}
	return getFileDetails(goListJSON)
}

func getFileDetails(goListJSON *goListJSON) (testFileDetailsByTest, error) {
	testFileDetailByTest := map[string]*testFileDetail{}
	for _, file := range goListJSON.TestGoFiles {
		sourceFilePath := fmt.Sprintf("%s/%s", goListJSON.Dir, file)
		fileSet := token.NewFileSet()
		f, err := parser.ParseFile(fileSet, sourceFilePath, nil, 0)
		if err != nil {
			return nil, err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				testFileDetail := &testFileDetail{}
				fileSetPos := fileSet.Position(n.Pos())
				folders := strings.Split(fileSetPos.String(), "/")
				fileNameWithPos := folders[len(folders)-1]
				fileDetails := strings.Split(fileNameWithPos, ":")
				lineNum, _ := strconv.Atoi(fileDetails[1])
				colNum, _ := strconv.Atoi(fileDetails[2])
				testFileDetail.FileName = fileDetails[0]
				testFileDetail.TestFunctionFilePos = testFunctionFilePos{
					Line: lineNum,
					Col:  colNum,
				}
				testFileDetailByTest[x.Name.Name] = testFileDetail
			}
			return true
		})
	}
	return testFileDetailByTest, nil
}

type testRef struct {
	key  string
	name string
}
type byName []testRef

func (t byName) Len() int {
	return len(t)
}
func (t byName) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
func (t byName) Less(i, j int) bool {
	return t[i].name < t[j].name
}

func generateReport(tmplData *templateData, allTests map[string]*testStatus, testFileDetailByPackage testFileDetailsByPackage, elapsedTestTime time.Duration, reportFileWriter *bufio.Writer, outptuEnvFile string) error {
	// // read the html template from the generated embedded asset go file
	// testReportHTMLTemplateStr, err := ioutil.ReadFile("../dist/report.html.template")
	tpl := template.New("report.html.template")
	tpl, err := tpl.Parse(string(testReportHTMLTemplateStr))
	if err != nil {
		return err
	}
	// testReportJsCodeStr, err := ioutil.ReadFile("../dist/report.js")
	// if err != nil {
	// 	log.Panicf("failed reading data from file: %s", err)
	// }
	tmplData.NumOfTestPassed = 0
	tmplData.NumOfTestFailed = 0
	tmplData.NumOfTestSkipped = 0
	tmplData.JsCode = template.JS(testReportJsCodeStr)
	tgCounter := 0
	tgID := 0

	// sort the allTests map by test name (this will produce a consistent order when iterating through the map)
	var tests []testRef
	for test, status := range allTests {
		tests = append(tests, testRef{test, status.TestName})
	}

	sort.Sort(byName(tests))
	for _, test := range tests {
		status := allTests[test.key]
		if len(tmplData.TestResults) == tgID {
			tmplData.TestResults = append(tmplData.TestResults, &testGroupData{})
		}
		// add file info(name and position; line and col) associated with the test function
		testFileInfo := testFileDetailByPackage[status.Package][status.TestName]
		if testFileInfo != nil {
			status.TestFileName = testFileInfo.FileName
			status.TestFunctionDetail = testFileInfo.TestFunctionFilePos
		}
		tmplData.TestResults[tgID].TestResults = append(tmplData.TestResults[tgID].TestResults, status)
		if !status.Passed {
			if !status.Skipped {
				tmplData.TestResults[tgID].FailureIndicator = "failed"
				tmplData.NumOfTestFailed++
			} else {
				tmplData.TestResults[tgID].SkippedIndicator = "skipped"
				tmplData.NumOfTestSkipped++
			}
		} else {
			tmplData.NumOfTestPassed++
		}
		tgCounter++
	}
	tmplData.NumOfTests = tmplData.NumOfTestPassed + tmplData.NumOfTestFailed + tmplData.NumOfTestSkipped
	status := Status{}
	status.Total = tmplData.NumOfTests
	status.Pass = tmplData.NumOfTestPassed
	status.Skip = tmplData.NumOfTestSkipped
	status.Fail = tmplData.NumOfTestFailed

	writeStatus(&status, outptuEnvFile)

	tmplData.TestDuration = elapsedTestTime.Round(time.Millisecond)
	td := time.Now()
	tmplData.TestExecutionDate = fmt.Sprintf("%s %d, %d %02d:%02d:%02d",
		td.Month(), td.Day(), td.Year(), td.Hour(), td.Minute(), td.Second())
	status.ElapsedTime = tmplData.TestExecutionDate
	writeStatus(&status, outptuEnvFile)
	if err := tpl.Execute(reportFileWriter, tmplData); err != nil {
		return err
	}
	return nil
}

func parseSizeFlag(tmplData *templateData, flags *cmdFlags) error {
	flags.sizeFlag = strings.ToLower(flags.sizeFlag)
	if !strings.Contains(flags.sizeFlag, "x") {
		val, err := strconv.Atoi(flags.sizeFlag)
		if err != nil {
			return err
		}
		tmplData.TestResultGroupIndicatorWidth = fmt.Sprintf("%dpx", val)
		tmplData.TestResultGroupIndicatorHeight = fmt.Sprintf("%dpx", val)
		return nil
	}
	if strings.Count(flags.sizeFlag, "x") > 1 {
		return errors.New(`malformed size value; only one x is allowed if specifying with and height`)
	}
	a := strings.Split(flags.sizeFlag, "x")
	valW, err := strconv.Atoi(a[0])
	if err != nil {
		return err
	}
	tmplData.TestResultGroupIndicatorWidth = fmt.Sprintf("%dpx", valW)
	valH, err := strconv.Atoi(a[1])
	if err != nil {
		return err
	}
	tmplData.TestResultGroupIndicatorHeight = fmt.Sprintf("%dpx", valH)
	return nil
}

func checkIfStdinIsPiped() error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return nil
	}
	return errors.New("ERROR: missing ≪ stdin ≫ pipe")
}

func writeStatus(status *Status, outptuEnvFile string) {
	content := []byte(fmt.Sprintf("export TOTAL=%d\nexport PASS=%d\nexport FAIL=%d\nexport SKIP=%d\n", status.Total, status.Pass, status.Fail, status.Skip))
	_ = ioutil.WriteFile(outptuEnvFile, content, 0644)
}
