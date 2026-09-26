package handler

// TODO: Handler tests need pi testing infrastructure.
// pi.Context cannot be created externally, so handler tests
// require either:
// 1. A TestContext helper in the pi package
// 2. Starting a real pi server via httptest
// 3. Testing through integration tests
//
// The handler signatures are now: func(ctx *pi.Context) (any, error)
// which can only be invoked through pi's internal routing.
