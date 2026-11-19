// Package errcore is the error engine.
// Application codes and client messages live in internal/errs.
// Startup calls errs.MustLoad and errcore.SetDebug. This package does not register that catalog.
// Add a new error in internal/errs. This package stays unchanged.
package errcore
