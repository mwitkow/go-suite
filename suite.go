package suite

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"
)

var matchMethod = flag.String("testify.m", "", "regular expression to select tests of the testify suite to run")

// Suite is a basic testing suite with methods for storing and
// retrieving the current *testing.T context.
type Suite struct {
	t *testing.T
}

// T retrieves the current *testing.T context.
func (suite *Suite) T() *testing.T {
	return suite.t
}

// SetT sets the current *testing.T context.
func (suite *Suite) SetT(t *testing.T) {
	suite.t = t
}

// Run takes a testing suite and runs all of the tests attached
// to it.
func Run(suiteT *testing.T, suite TestingSuite) {
	suite.SetT(suiteT)

	if setupAllSuite, ok := suite.(SetupAllSuite); ok {
		setupAllSuite.SetupSuite()
	}
	defer func() {
		suite.SetT(suiteT)
		if tearDownAllSuite, ok := suite.(TearDownAllSuite); ok {
			tearDownAllSuite.TearDownSuite()
		}
	}()

	methodFinder := reflect.TypeOf(suite)
	for index := 0; index < methodFinder.NumMethod(); index++ {
		method := methodFinder.Method(index)
		ok, err := methodFilter(method.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "testify: invalid regexp for -m: %s\n", err)
			os.Exit(1)
		}
		if ok {
			suiteT.Run(method.Name, func(testT *testing.T){
				suite.SetT(testT)
				if setupTestSuite, ok := suite.(SetupTestSuite); ok {
					setupTestSuite.SetupTest()
				}
				if beforeTestSuite, ok := suite.(BeforeTest); ok {
					// This is legacy behaviour that calls the test by the struct name and not the test name.
					beforeTestSuite.BeforeTest(methodFinder.Elem().Name(), method.Name)
				}
				defer func() {
					if afterTestSuite, ok := suite.(AfterTest); ok {
						afterTestSuite.AfterTest(methodFinder.Elem().Name(), method.Name)
					}
					if tearDownTestSuite, ok := suite.(TearDownTestSuite); ok {
						// This is legacy behaviour that calls the test by the struct name and not the test name.
						tearDownTestSuite.TearDownTest()
					}
					suite.SetT(suiteT)
				}()
				if method.Type.NumIn() == 1 {
					method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
				} else {
					testT.Fatalf("suite: too many arguments to method %v", method.Name)
				}
			})
			suite.SetT(suiteT)
		}
	}
}

// Filtering method according to set regular expression
// specified command-line argument -m
func methodFilter(name string) (bool, error) {
	if ok, _ := regexp.MatchString("^Test", name); !ok {
		return false, nil
	}
	return regexp.MatchString(*matchMethod, name)
}
