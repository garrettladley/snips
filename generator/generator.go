package generator

type GenerateOpt func(g *generator) error

type generator struct{}

func Generate(opts ...GenerateOpt) (literals string, err error) {
	g := generator{}
	for _, opt := range opts {
		if err := opt(&g); err != nil {
			return "", err
		}
	}
	return "", nil
}
