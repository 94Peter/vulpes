package csv

type Option func(*Config)

func WithBom(enable bool) Option {
	return func(c *Config) {
		c.UseBOM = enable
	}
}

func WithCRLF(enable bool) Option {
	return func(c *Config) {
		c.UseCRLF = enable
	}
}

func WithDelimiter(delimiter rune) Option {
	return func(c *Config) {
		c.Delimiter = delimiter
	}
}
