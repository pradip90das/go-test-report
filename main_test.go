package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, _, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"version"})
	rootCmdErr := rootCmd.Execute()
	assertions.Nil(rootCmdErr)
	output, readErr := ioutil.ReadAll(buffer)
	assertions.Nil(readErr)
	assertions.Equal(fmt.Sprintf("go-test-report v%s\n", version), string(output))
}

func TestTitleFlag(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, tmplData, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--title", "Sample Test Report"})
	rootCmdErr := rootCmd.Execute()
	assertions.Error(rootCmdErr)
	output, readErr := ioutil.ReadAll(buffer)
	assertions.Nil(readErr)
	assertions.Equal("Sample Test Report", tmplData.ReportTitle)
	assertions.NotEmpty(output)
}

func TestTitleFlagIfMissingValue(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, _, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--title"})
	rootCmdErr := rootCmd.Execute()
	assertions.NotNil(rootCmdErr)
	assertions.Equal(rootCmdErr.Error(), `flag needs an argument: --title`)
}

func TestSizeFlag(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, tmplData, flags := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--size", "24"})
	rootCmdErr := rootCmd.Execute()
	assertions.Error(rootCmdErr)
	output, readErr := ioutil.ReadAll(buffer)
	assertions.Nil(readErr)
	assertions.Equal("24", flags.sizeFlag)
	assertions.Equal("24px", tmplData.TestResultGroupIndicatorWidth)
	assertions.Equal("24px", tmplData.TestResultGroupIndicatorHeight)
	assertions.NotEmpty(output)
}

func TestSizeFlagWithFullDimensions(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, tmplData, flags := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--size", "24x16"})
	rootCmdErr := rootCmd.Execute()
	assertions.Error(rootCmdErr)
	output, readErr := ioutil.ReadAll(buffer)
	assertions.Nil(readErr)
	assertions.Equal("24x16", flags.sizeFlag)
	assertions.Equal("24px", tmplData.TestResultGroupIndicatorWidth)
	assertions.Equal("16px", tmplData.TestResultGroupIndicatorHeight)
	assertions.NotEmpty(output)
}

func TestSizeFlagIfMissingValue(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, _, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--size"})
	rootCmdErr := rootCmd.Execute()
	assertions.NotNil(rootCmdErr)
	assertions.Equal(rootCmdErr.Error(), `flag needs an argument: --size`)
}

func TestGroupSizeFlag(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, tmplData, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--groupSize", "32"})
	rootCmdErr := rootCmd.Execute()
	assertions.Error(rootCmdErr)
	output, readErr := ioutil.ReadAll(buffer)
	assertions.Nil(readErr)
	assertions.Equal(32, tmplData.numOfTestsPerGroup)
	assertions.NotEmpty(output)
}

func TestGroupSizeFlagIfMissingValue(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, _, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--groupSize"})
	rootCmdErr := rootCmd.Execute()
	assertions.NotNil(rootCmdErr)
	assertions.Equal(rootCmdErr.Error(), `flag needs an argument: --groupSize`)
}

func TestGroupOutputFlag(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, tmplData, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--output", "test_file.html"})
	rootCmdErr := rootCmd.Execute()
	assertions.Error(rootCmdErr)
	output, readErr := ioutil.ReadAll(buffer)
	assertions.Nil(readErr)
	assertions.Equal("test_file.html", tmplData.OutputFilename)
	assertions.NotEmpty(output)
}

func TestGroupOutputFlagIfMissingValue(t *testing.T) {
	assertions := assert.New(t)
	buffer := bytes.NewBufferString("")
	rootCmd, _, _ := initRootCommand()
	rootCmd.SetOut(buffer)
	rootCmd.SetArgs([]string{"--output"})
	rootCmdErr := rootCmd.Execute()
	assertions.NotNil(rootCmdErr)
	assertions.Equal(rootCmdErr.Error(), `flag needs an argument: --output`)
}

func TestGetAllDetails(t *testing.T) {
	assertions := assert.New(t)
	data := `{
	"Dir": ".",
	"ImportPath": "github.com/vakenbolt/go-test-report",
	"Name": "main",
	"Module": {
		"Path": "github.com/vakenbolt/go-test-report",
		"Main": true,
		"Dir": "."
	},
	"GoFiles": [
		"main.go"
	],
	"TestGoFiles": [
		"main_test.go"
	]
}`
	tmpfile, err := ioutil.TempFile("", "*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err = tmpfile.WriteString(data); err != nil {
		t.Fatal(err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	testFileDetailsByPackage, err := getAllDetails(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	assertions.Len(testFileDetailsByPackage, 1)
}

func TestGenerateReport(t *testing.T) {
	assertions := assert.New(t)
	tmplData := &templateData{
		TestResultGroupIndicatorWidth:  "20px",
		TestResultGroupIndicatorHeight: "16px",
		ReportTitle:                    "test-title",
		numOfTestsPerGroup:             2,
		OutputFilename:                 "test-output-report.html",
	}
	allTests := map[string]*testStatus{}
	allTests["TestFunc1"] = &testStatus{
		TestName:           "TestFunc1",
		Package:            "go-test-report",
		ElapsedTime:        0,
		Output:             nil,
		Passed:             true,
		TestFileName:       "",
		TestFunctionDetail: testFunctionFilePos{},
	}
	allTests["TestFunc2"] = &testStatus{
		TestName:     "TestFunc2",
		Package:      "package2",
		ElapsedTime:  0,
		Output:       nil,
		Passed:       true,
		TestFileName: "",

		TestFunctionDetail: testFunctionFilePos{},
	}
	allTests["TestFunc3"] = &testStatus{
		TestName:           "TestFunc3",
		Package:            "go-test-report",
		ElapsedTime:        0,
		Output:             nil,
		Passed:             false,
		TestFileName:       "",
		TestFunctionDetail: testFunctionFilePos{},
	}
	allTests["TestFunc4"] = &testStatus{
		TestName:           "TestFunc4",
		Package:            "go-test-report",
		ElapsedTime:        0,
		Output:             nil,
		Passed:             false,
		Skipped:            true,
		TestFileName:       "",
		TestFunctionDetail: testFunctionFilePos{},
	}
	testFileDetailsByPackage := testFileDetailsByPackage{}
	testFileDetailsByPackage["go-test-report"] = map[string]*testFileDetail{}
	testFileDetailsByPackage["go-test-report"]["TestFunc1"] = &testFileDetail{
		FileName: "sample_file_1.go",
		TestFunctionFilePos: testFunctionFilePos{
			Line: 101,
			Col:  1,
		},
	}
	testFileDetailsByPackage["package2"] = map[string]*testFileDetail{}
	testFileDetailsByPackage["package2"]["TestFunc2"] = &testFileDetail{
		FileName: "sample_file_2.go",
		TestFunctionFilePos: testFunctionFilePos{
			Line: 784,
			Col:  17,
		},
	}
	testFileDetailsByPackage["go-test-report"]["TestFunc3"] = &testFileDetail{
		TestFunctionFilePos: testFunctionFilePos{
			Line: 0,
			Col:  0,
		},
	}
	elapsedTestTime := 3 * time.Second
	writer := bufio.NewWriter(&bytes.Buffer{})
	err := generateReport(tmplData, allTests, testFileDetailsByPackage, elapsedTestTime, writer)
	assertions.Nil(err)
	assertions.Equal(2, tmplData.NumOfTestPassed)
	assertions.Equal(1, tmplData.NumOfTestFailed)
	assertions.Equal(1, tmplData.NumOfTestSkipped)
	assertions.Equal(4, tmplData.NumOfTests)

	assertions.Equal("TestFunc1", tmplData.TestResults[0].TestResults[0].TestName)
	assertions.Equal("go-test-report", tmplData.TestResults[0].TestResults[0].Package)
	assertions.Equal(true, tmplData.TestResults[0].TestResults[0].Passed)
	assertions.Equal("sample_file_1.go", tmplData.TestResults[0].TestResults[0].TestFileName)
	assertions.Equal(1, tmplData.TestResults[0].TestResults[0].TestFunctionDetail.Col)
	assertions.Equal(101, tmplData.TestResults[0].TestResults[0].TestFunctionDetail.Line)

	assertions.Equal("TestFunc2", tmplData.TestResults[0].TestResults[1].TestName)
	assertions.Equal("package2", tmplData.TestResults[0].TestResults[1].Package)
	assertions.Equal(true, tmplData.TestResults[0].TestResults[1].Passed)
	assertions.Equal("sample_file_2.go", tmplData.TestResults[0].TestResults[1].TestFileName)
	assertions.Equal(17, tmplData.TestResults[0].TestResults[1].TestFunctionDetail.Col)
	assertions.Equal(784, tmplData.TestResults[0].TestResults[1].TestFunctionDetail.Line)

	assertions.Equal("TestFunc3", tmplData.TestResults[1].TestResults[0].TestName)
	assertions.Equal("go-test-report", tmplData.TestResults[1].TestResults[0].Package)
	assertions.Equal(false, tmplData.TestResults[1].TestResults[0].Passed)
	assertions.Empty(tmplData.TestResults[1].TestResults[0].TestFileName)
	assertions.Equal(0, tmplData.TestResults[1].TestResults[0].TestFunctionDetail.Col)
	assertions.Equal(0, tmplData.TestResults[1].TestResults[0].TestFunctionDetail.Line)
}

func TestParseSizeFlagIfValueIsNotInteger(t *testing.T) {
	assertions := assert.New(t)
	tmplData := &templateData{}
	flags := &cmdFlags{
		sizeFlag: "x",
	}
	err := parseSizeFlag(tmplData, flags)
	assertions.Error(err)
	assertions.Equal(err.Error(), `strconv.Atoi: parsing "": invalid syntax`)

}

func TestParseSizeFlagIfWidthValueIsNotInteger(t *testing.T) {
	assertions := assert.New(t)
	tmplData := &templateData{}
	flags := &cmdFlags{
		sizeFlag: "Bx27",
	}
	err := parseSizeFlag(tmplData, flags)
	assertions.Error(err)
	assertions.Equal(err.Error(), `strconv.Atoi: parsing "b": invalid syntax`)
}

func TestParseSizeFlagIfHeightValueIsNotInteger(t *testing.T) {
	assertions := assert.New(t)
	tmplData := &templateData{}
	flags := &cmdFlags{
		sizeFlag: "10xA",
	}
	err := parseSizeFlag(tmplData, flags)
	assertions.Error(err)
	assertions.Equal(err.Error(), `strconv.Atoi: parsing "a": invalid syntax`)
}

func TestParseSizeFlagIfMalformedSize(t *testing.T) {
	assertions := assert.New(t)
	tmplData := &templateData{}
	flags := &cmdFlags{
		sizeFlag: "10xx19",
	}
	err := parseSizeFlag(tmplData, flags)
	assertions.Error(err)
	assertions.Equal(err.Error(), `malformed size value; only one x is allowed if specifying with and height`)
}
