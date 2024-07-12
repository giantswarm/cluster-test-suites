// package timeout contains types and constants to be used when overriding default test timeouts
//
// Each test case that supports overriding the timeout it uses by default will need its own `TestKey` defining
// in the constants. Once this is available it can be used within the test case like the following:
//
//	timeout := state.GetTestTimeout(timeout.DeployApps, 15*time.Minute)
//
// To then override the timeout in a specific test sutie you can do so like this following:
//
//	state.SetTestTimeout(timeout.DeployApps, time.Minute*25)
package timeout
