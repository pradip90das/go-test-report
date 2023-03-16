/**
 * @typedef TestStatus
 * @property {string} TestName
 * @property {string} Package
 * @property {number} ElapsedTime
 * @property {Array.<string>} Output
 * @property {Array.<string>} Screenshots
 * @property {boolean} Passed
 * @property {boolean} Skipped
 */
class TestStatus { }

/**
 * @typedef TestGroupData
 * @type {object}
 * @property {string} FailureIndicator
 * @property {string} SkippedIndicator
 * @property {Array.<TestStatus>}
 */
class TestGroupData { }

/**
 * @typedef TestResults
 * @type {Array.<TestGroupData>}
 */
class TestResults extends Array { }

/**
 * @typedef SelectedItems
 * @property {HTMLElement|EventTarget} testResults
 * @property {String} selectedTestGroupColor
 */
class SelectedItems { }

/**
 * @typedef GoTestReportElements
 * @property {TestResults} data
 * @property {HTMLElement} testResultsElem
 * @property {HTMLElement} testGroupListElem
 */
class GoTestReportElements { }


/**
 * Main entry point for GoTestReport.
 * @param {GoTestReportElements} elements
 * @returns {{testResultsClickHandler: testResultsClickHandler}}
 * @constructor
 */
var testCaseFilter = ["PASS", "FAIL", "SKIP"]

function Filter(status) {
  testCaseFilter = status
  document.getElementById('0').click()
}

window.GoTestReport = function (elements) {
  const /**@type {SelectedItems}*/ selectedItems = {
    testResults: null,
    selectedTestGroupColor: null
  }

  function addEventData(event) {
    if (event.data == null) {
      event.data = { target: event.target }
    }
    return event
  }
  const goTestReport = {
    /**
     * Invoked when a user clicks on one of the test group div elements.
     * @param {HTMLElement} target The element associated with the test group.
     * @param {boolean} shiftKey If pressed, all of test detail associated to the test group is shown.
     * @param {TestResults} data
     * @param {SelectedItems} selectedItems
     * @param {function(target: Element, data: TestResults)} testGroupListHandler
     */
    testResultsClickHandler: function (target,
      shiftKey,
      data,
      selectedItems,
      testGroupListHandler) {

      if (target.classList.contains('testResultGroup') === false) {
        return
      }
      if (selectedItems.testResults != null) {
        let testResultsElement = /**@type {HTMLElement}*/ selectedItems.testResults
        testResultsElement.classList.remove("selected")
        testResultsElement.style.backgroundColor = selectedItems.selectedTestGroupColor
      }
      const testGroupId = /**@type {number}*/ target.id
      if ((target.id === undefined)
        || (data[testGroupId] === undefined)
        || (data[testGroupId]['TestResults'] === undefined)) {
        return
      }
      const testResults = /**@type {TestResults}*/ data[testGroupId]['TestResults']
      let testGroupList = /**@type {string}*/ ''
      selectedItems.selectedTestGroupColor = getComputedStyle(target).getPropertyValue('background-color')
      selectedItems.testResults = target
      target.classList.add("selected")
      for (let i = 0; i < testResults.length; i++) {
        const testResult = /**@type {TestGroupData}*/ testResults[i]
        const testPassed = /**@type {boolean}*/ testResult.Passed
        const testSkipped = /**@type {boolean}*/ testResult.Skipped
        const testPassedStatus = /**@type {string}*/ (testPassed) ? '' : (testSkipped ? 'skipped' : 'failed')
        const testStatus = /**@type {string}*/ (testPassed) ? 'PASS' : (testSkipped ? 'SKIP' : 'FAIL')
        const testId = /**@type {string}*/ target.attributes['id'].value
        if (testCaseFilter == undefined || testCaseFilter.includes(testStatus)) {
          testGroupList += `<div id=${testResult.TestName} class="testGroupRow ${testPassedStatus}" data-groupid="${testId}" data-index="${i}">
        <span class="testTextStatus ${testPassedStatus}">${testStatus}</span>
        <span class="testStatus ${testPassedStatus}">${(testPassed) ? '&check' : (testSkipped ? '&dash' : '&cross')};</span>
        <span class="testTitle">${testResult.TestName}</span>
        <span class="testDuration"><span>${testResult.ElapsedTime}s </span>⏱</span>
      </div>`
        }

      }
      if (testGroupList === '') {
        testGroupList += `<div <div style="padding-top: 50px;margin-left: 45%;"><span class="">No ${testCaseFilter} testcase</span></div>`
      }
      const testGroupListElem = elements.testGroupListElem
      testGroupListElem.innerHTML = ''
      testGroupListElem.innerHTML = testGroupList

      if (shiftKey) {
        testGroupListElem.querySelectorAll('.testGroupRow')
          .forEach((elem) => testGroupListHandler(elem, data))
      } else if (testResults.length === 1) {
        testGroupListHandler(testGroupListElem.querySelector('.testGroupRow'), data)
      }
    },

    /**
     *
     * @param {Element} target
     * @param {TestResults} data
     */
    testGroupListHandler: function (target, data) {
      const attribs = target['attributes']
      if (attribs.hasOwnProperty('data-groupid')) {
        const groupId = /**@type {number}*/ attribs['data-groupid'].value
        const testIndex = /**@type {number}*/ attribs['data-index'].value
        const testStatus = /**@type {TestStatus}*/ data[groupId]['TestResults'][testIndex]
        const testOutputDiv = /**@type {HTMLDivElement}*/ target.querySelector('div.testOutput')

        if (testOutputDiv == null) {
          const testOutputDiv = document.createElement('div')
          testOutputDiv.classList.add('testOutput')
          const consolePre = document.createElement('pre')
          consolePre.classList.add('console')
          const testDetailDiv = document.createElement('div')
          testDetailDiv.classList.add('testDetail')
          const packageNameDiv = document.createElement('div')
          packageNameDiv.classList.add('package')
          packageNameDiv.innerHTML = `<strong>Package:</strong> ${testStatus.Package}`
          const screenshotDiv = document.createElement('div')
          screenshotDiv.classList.add('package')
          if (testStatus.Screenshots && testStatus.Screenshots.length > 0) {
            const ul = document.createElement('ul')
            testStatus.Screenshots.forEach(element => {
              var li = document.createElement('li');
              var link = document.createElement('a');
              link.setAttribute("href", element);
              link.setAttribute("target", "_blank");
              var linkText = document.createTextNode(element);
              link.appendChild(linkText);
              li.appendChild(link);
              ul.appendChild(li);
            });
            screenshotDiv.innerHTML = `<strong>Screenshots:</strong>`
            screenshotDiv.append(ul)
          }
          const testFileNameDiv = document.createElement('div')
          testFileNameDiv.classList.add('filename')
          if (testStatus.TestFileName.trim() === "") {
            testFileNameDiv.innerHTML = `<strong>Filename:</strong> n/a &nbsp;&nbsp;`
          } else {
            testFileNameDiv.innerHTML = `<strong>Filename:</strong> ${testStatus.TestFileName} &nbsp;&nbsp;`
            testFileNameDiv.innerHTML += `<strong>Line:</strong> ${testStatus.TestFunctionDetail.Line} `
            testFileNameDiv.innerHTML += `<strong>Col:</strong> ${testStatus.TestFunctionDetail.Col}`
          }
          testDetailDiv.insertAdjacentElement('beforeend', screenshotDiv)
          testDetailDiv.insertAdjacentElement('beforeend', packageNameDiv)
          testDetailDiv.insertAdjacentElement('beforeend', testFileNameDiv)
          testOutputDiv.insertAdjacentElement('afterbegin', consolePre)
          testOutputDiv.insertAdjacentElement('beforeend', testDetailDiv)
          target.insertAdjacentElement('beforeend', testOutputDiv)

          if (testStatus.Passed) {
            consolePre.classList.remove('skipped')
            consolePre.classList.remove('failed')
          } else if (testStatus.Skipped) {
            consolePre.classList.add('skipped')
            consolePre.classList.remove('failed')
          } else {
            consolePre.classList.remove('skipped')
            consolePre.classList.add('failed')
          }
          consolePre.textContent = testStatus.Output.join('')
        } else {
          testOutputDiv.remove()
        }
      }
    }
  }

  //+------------------------+
  //|    setup DOM events    |
  //+------------------------+
  elements.testResultsElem
    .addEventListener('click', event =>
      goTestReport.testResultsClickHandler(/**@type {HTMLElement}*/ addEventData(event).data.target,
        event.shiftKey,
        elements.data,
        selectedItems,
        goTestReport.testGroupListHandler))

  elements.testGroupListElem
    .addEventListener('click', event =>
      goTestReport.testGroupListHandler(/**@type {Element}*/ event.target,
        elements.data))
  window.onload = function () {
    const queryString = window.location.search;
    const urlParams = new URLSearchParams(queryString);
    const testcase = urlParams.get('testcase')
    if (testcase) {
      target = document.getElementById(testcase);
      goTestReport.testGroupListHandler(/**@type {Element}*/ target,
        elements.data)
      target.scrollIntoView();
    }
  };
  return goTestReport
}
