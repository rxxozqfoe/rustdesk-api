package config

type S3 struct {
	Endpoint  string `mapstructure:"endpoint"` // e.g. "minio:9000" or "s3.amazonaws.com"
	AccessKey string `mapstructure:"access-key"`
	SecretKey string `mapstructure:"secret-key"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	UseSSL    bool   `mapstructure:"use-ssl"`
}

func (s *S3) Enabled() bool {
	return s.Endpoint != "" && s.Bucket != ""
}
