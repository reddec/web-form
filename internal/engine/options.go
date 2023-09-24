package engine

type FormOption func(config *FormConfig)

func WithXSRF(enable bool) FormOption {
	return func(config *FormConfig) {
		config.XSRF = enable
	}
}
