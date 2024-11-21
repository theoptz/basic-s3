package weight

type DistributorConfig struct {
	Endpoints   []string
	Weights     []int
	MaxParts    int
	MinPartSize int
}
