package weight

import (
	"fmt"
	"math/rand"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/theoptz/basic-s3/proto"
)

type WeightDistributor struct {
	clients     []proto.StorageClient
	weights     []int
	minPartSize int
	maxParts    int
}

func (w *WeightDistributor) GetPlan(fileSize int) ([]int, int) {
	if fileSize <= 0 {
		return nil, 0
	}

	var parts int
	size := fileSize / w.maxParts
	if size >= w.minPartSize {
		parts = w.maxParts
	} else {
		size = w.minPartSize
		parts = fileSize / w.minPartSize
		if fileSize%w.minPartSize != 0 {
			parts++
		}
	}

	selectedServers := selectServers(w.weights, parts)

	return selectedServers, size
}

func (w *WeightDistributor) GetClientByID(id int) (proto.StorageClient, error) {
	if id >= len(w.clients) {
		return nil, fmt.Errorf("client %d not found", id)
	}

	return w.clients[id], nil
}

func New(cfg DistributorConfig) (*WeightDistributor, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints provided")
	}

	if len(cfg.Weights) != len(cfg.Endpoints) {
		return nil, fmt.Errorf("invalid weights")
	}

	w := &WeightDistributor{
		clients:     make([]proto.StorageClient, len(cfg.Endpoints)),
		weights:     cfg.Weights,
		maxParts:    cfg.MaxParts,
		minPartSize: cfg.MinPartSize,
	}

	for i := range cfg.Endpoints {
		conn, err := grpc.NewClient(cfg.Endpoints[i], grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}

		w.clients[i] = proto.NewStorageClient(conn)
	}

	return w, nil
}

func selectServers(weights []int, n int) []int {
	availableServers := make([]int, len(weights))
	copy(availableServers, weights)

	selectedServers := make([]int, 0, n)

	totalWeight := 0
	for _, weight := range availableServers {
		totalWeight += weight
	}

	for i := 0; i < n; i++ {
		randomValue := rand.Intn(totalWeight) + 1

		cumulativeWeight := 0
		for index, weight := range availableServers {
			cumulativeWeight += weight
			if randomValue <= cumulativeWeight {
				selectedServers = append(selectedServers, index)

				totalWeight -= weight
				availableServers[index] = 0
				break
			}
		}
	}

	return selectedServers
}
