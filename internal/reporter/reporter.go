package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/meomkarjagtap/neukeiho-agent/internal/metrics"
)

// Reporter sends metric snapshots to the NeuKeiho controller.
type Reporter struct {
	nodeID     string
	controllerURL string
	client     *http.Client
}

// New creates a new Reporter.
func New(nodeID, host, port string) *Reporter {
	return &Reporter{
		nodeID:        nodeID,
		controllerURL: fmt.Sprintf("http://%s:%s/metrics", host, port),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type payload struct {
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
	CPU       float64   `json:"cpu_percent"`
	Memory    float64   `json:"memory_percent"`
	Disk      float64   `json:"disk_percent"`
	NetworkRx float64   `json:"network_rx_mbps"`
	NetworkTx float64   `json:"network_tx_mbps"`
}

// Push sends a snapshot to the controller.
func (r *Reporter) Push(s metrics.Snapshot) error {
	p := payload{
		NodeID:    r.nodeID,
		Timestamp: s.Timestamp,
		CPU:       s.CPU,
		Memory:    s.Memory,
		Disk:      s.Disk,
		NetworkRx: s.NetworkRx,
		NetworkTx: s.NetworkTx,
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := r.client.Post(r.controllerURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("controller returned %d", resp.StatusCode)
	}

	return nil
}
