package config

// CustomClient holds config for the custom client feature.
// Build execution is handled by the build-worker; the API server only manages
// records, signing, and S3/worker proxying.
type CustomClient struct{}

func (cc *CustomClient) Init() {}
