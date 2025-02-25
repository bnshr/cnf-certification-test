package failures

import (
	"fmt"
	"testing"

	"github.com/test-network-function/cnf-certification-test/cmd/tnf/pkg/claim"
	"gotest.tools/v3/assert"
)

func TestParseTargetTestSuitesFlag(t *testing.T) {
	testCases := []struct {
		flag               string
		expectedTestSuites map[string]bool
	}{
		{
			flag:               "",
			expectedTestSuites: nil,
		},
		{
			flag: "test-suite1",
			expectedTestSuites: map[string]bool{
				"test-suite1": true,
			},
		},
		{
			flag: "test-suite1,test-suite2",
			expectedTestSuites: map[string]bool{
				"test-suite1": true,
				"test-suite2": true,
			},
		},
		{
			flag: "test-suite1 , test-suite2  ",
			expectedTestSuites: map[string]bool{
				"test-suite1": true,
				"test-suite2": true,
			},
		},
		{
			flag: "test-suite1 , test-suite2,test-suite-3  ",
			expectedTestSuites: map[string]bool{
				"test-suite1":  true,
				"test-suite2":  true,
				"test-suite-3": true,
			},
		},
	}

	for _, tc := range testCases {
		testSuitesFlag = tc.flag
		parsedTestSuites := parseTargetTestSuitesFlag()
		assert.DeepEqual(t, tc.expectedTestSuites, parsedTestSuites)
	}
}

func TestParseOutputFormatFlag(t *testing.T) {
	testCases := []struct {
		flag           string
		expectedFormat string
		expectedError  string
	}{
		{
			flag:          "",
			expectedError: `invalid output format flag "" - available formats: [text json]`,
		},
		{
			flag:          "invalid-format",
			expectedError: `invalid output format flag "invalid-format" - available formats: [text json]`,
		},
		{
			flag:           "text",
			expectedFormat: "text",
		},
		{
			flag:           "json",
			expectedFormat: "json",
		},
	}

	for _, tc := range testCases {
		outputFormatFlag = tc.flag
		parsedFormat, err := parseOutputFormatFlag()
		if err != nil {
			assert.Equal(t, tc.expectedError, err.Error())
		}
		assert.Equal(t, tc.expectedFormat, parsedFormat)
	}
}

func TestGetNonCompliantObjectsFromFailureReason(t *testing.T) {
	testCases := []struct {
		failureReason               string
		expectedNonCompliantObjects []NonCompliantObject
		expectedError               string
	}{
		{
			failureReason:               "",
			expectedNonCompliantObjects: nil,
			expectedError:               `failed to decode failureReason : unexpected end of JSON input`,
		},
		{
			failureReason:               `{"CompliantObjectsOut": null, "NonCompliantObjectsOut": null}`,
			expectedNonCompliantObjects: []NonCompliantObject{},
		},
		// One container failed the SYS_ADMIN check.
		{
			failureReason: `{
				"CompliantObjectsOut": null,
				"NonCompliantObjectsOut": [
				  {
					"ObjectType": "Container",
					"ObjectFieldsKeys": [
					  "Reason For Non Compliance",
					  "Namespace",
					  "Pod Name",
					  "Container Name",
					  "SCC Capability"
					],
					"ObjectFieldsValues": [
					  "Non compliant capability detected in container",
					  "tnf",
					  "test-887998557-8gwwm",
					  "test",
					  "SYS_ADMIN"
					]
				  }
				]
			}`,
			expectedNonCompliantObjects: []NonCompliantObject{
				{
					Type:   "Container",
					Reason: "Non compliant capability detected in container",
					Spec: ObjectSpec{
						Fields: []struct {
							Key   string
							Value string
						}{
							{
								Key:   "Namespace",
								Value: "tnf",
							},
							{
								Key:   "Pod Name",
								Value: "test-887998557-8gwwm",
							},
							{
								Key:   "Container Name",
								Value: "test",
							},
							{
								Key:   "SCC Capability",
								Value: "SYS_ADMIN",
							},
						},
					},
				},
			},
		},
		// Two containers failed the SYS_ADMIN check.
		{
			failureReason: `{
				"CompliantObjectsOut": null,
				"NonCompliantObjectsOut": [
				  {
					"ObjectType": "Container",
					"ObjectFieldsKeys": [
					  "Reason For Non Compliance",
					  "Namespace",
					  "Pod Name",
					  "Container Name",
					  "SCC Capability"
					],
					"ObjectFieldsValues": [
					  "Non compliant capability detected in container",
					  "tnf",
					  "test-887998557-8gwwm",
					  "test",
					  "SYS_ADMIN"
					]
				  },
				  {
					"ObjectType": "Container",
					"ObjectFieldsKeys": [
					  "Reason For Non Compliance",
					  "Namespace",
					  "Pod Name",
					  "Container Name",
					  "SCC Capability"
					],
					"ObjectFieldsValues": [
					  "Non compliant capability detected in container",
					  "tnf",
					  "test-887998557-pr2w5",
					  "test",
					  "SYS_ADMIN"
					]
				  }
				]
			  }
			`,
			expectedNonCompliantObjects: []NonCompliantObject{
				{
					Type:   "Container",
					Reason: "Non compliant capability detected in container",
					Spec: ObjectSpec{
						Fields: []struct {
							Key   string
							Value string
						}{
							{
								Key:   "Namespace",
								Value: "tnf",
							},
							{
								Key:   "Pod Name",
								Value: "test-887998557-8gwwm",
							},
							{
								Key:   "Container Name",
								Value: "test",
							},
							{
								Key:   "SCC Capability",
								Value: "SYS_ADMIN",
							},
						},
					},
				},
				{
					Type:   "Container",
					Reason: "Non compliant capability detected in container",
					Spec: ObjectSpec{
						Fields: []struct {
							Key   string
							Value string
						}{
							{
								Key:   "Namespace",
								Value: "tnf",
							},
							{
								Key:   "Pod Name",
								Value: "test-887998557-pr2w5",
							},
							{
								Key:   "Container Name",
								Value: "test",
							},
							{
								Key:   "SCC Capability",
								Value: "SYS_ADMIN",
							},
						},
					},
				},
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		nonCompliantObjects, err := getNonCompliantObjectsFromFailureReason(tc.failureReason)
		if err != nil {
			assert.Equal(t, tc.expectedError, err.Error())
		}

		assert.DeepEqual(t, tc.expectedNonCompliantObjects, nonCompliantObjects)
	}
}

// Uses claim files in testdata folder:
// claim1.json -> Two test suites, access-control & platform-alteration. One failed test case in the access-control ts.
// claim2.json -> Same as clam1.json, but the failureReason is a simple string, not using report objects yet.
func TestGetFailedTestCasesByTestSuite(t *testing.T) {
	testCases := []struct {
		claimFilePath            string
		targetTestSuite          string
		expectedFailedTestSuites []FailedTestSuite
	}{
		// Target test suite doesn't have any failed tc.
		{
			claimFilePath:            "testdata/claim1.json",
			targetTestSuite:          "platform-alteration",
			expectedFailedTestSuites: []FailedTestSuite{},
		},
		// Failed test case that doesn't use the report objects yet.
		{
			claimFilePath:   "testdata/claim2.json",
			targetTestSuite: "access-control",
			expectedFailedTestSuites: []FailedTestSuite{
				{
					TestSuiteName: "access-control",
					FailingTestCases: []FailedTestCase{
						{
							TestCaseName:        "access-control-sys-admin-capability-check",
							TestCaseDescription: "Ensures that containers do not use SYS_ADMIN capability",
							FailureReason:       "pod xxx ns yyy container zzz uses SYS_ADMIN",
						},
					},
				},
			},
		},
		{
			targetTestSuite: "access-control",
			claimFilePath:   "testdata/claim1.json",
			expectedFailedTestSuites: []FailedTestSuite{
				{
					TestSuiteName: "access-control",
					FailingTestCases: []FailedTestCase{
						{
							TestCaseName:        "access-control-sys-admin-capability-check",
							TestCaseDescription: "Ensures that containers do not use SYS_ADMIN capability",
							NonCompliantObjects: []NonCompliantObject{
								{
									Type:   "Container",
									Reason: "Non compliant capability detected in container",
									Spec: ObjectSpec{
										Fields: []struct {
											Key   string
											Value string
										}{
											{
												Key:   "Namespace",
												Value: "tnf",
											},
											{
												Key:   "Pod Name",
												Value: "test-887998557-8gwwm",
											},
											{
												Key:   "Container Name",
												Value: "test",
											},
											{
												Key:   "SCC Capability",
												Value: "SYS_ADMIN",
											},
										},
									},
								},
								{
									Type:   "Container",
									Reason: "Non compliant capability detected in container",
									Spec: ObjectSpec{
										Fields: []struct {
											Key   string
											Value string
										}{
											{
												Key:   "Namespace",
												Value: "tnf",
											},
											{
												Key:   "Pod Name",
												Value: "test-887998557-pr2w5",
											},
											{
												Key:   "Container Name",
												Value: "test",
											},
											{
												Key:   "SCC Capability",
												Value: "SYS_ADMIN",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		claimScheme, err := claim.Parse(tc.claimFilePath)
		assert.NilError(t, err)

		// Order test case results by test suite, using a helper map.
		resultsByTestSuite := map[string][]*claim.TestCaseResult{}
		for id := range claimScheme.Claim.Results {
			tcResult := claimScheme.Claim.Results[id][0]
			resultsByTestSuite[tcResult.TestID.Suite] = append(
				resultsByTestSuite[tcResult.TestID.Suite],
				&tcResult,
			)
		}

		testSuites := getFailedTestCasesByTestSuite(
			resultsByTestSuite,
			map[string]bool{tc.targetTestSuite: true},
		)
		fmt.Printf("%#v\n\n", testSuites)
		assert.DeepEqual(t, tc.expectedFailedTestSuites, testSuites)
	}
}
