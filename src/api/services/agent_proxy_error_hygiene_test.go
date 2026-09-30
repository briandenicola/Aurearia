package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const leakedProviderKey = "sk-live-leaked-provider-key"

var agentErrorBodyWithSecret = `{"detail":[{"loc":["body","llm","api_key"],"msg":"bad key","input":"` + leakedProviderKey + `"}],"api_key":"` + leakedProviderKey + `"}`

type agentProxyCall struct {
	name string
	call func(ctx context.Context, p *AgentProxy) error
}

func agentProxyCalls() []agentProxyCall {
	llm := LLMConfig{Provider: "anthropic", APIKey: leakedProviderKey, Model: "test"}
	return []agentProxyCall{
		{"AnalyzeCoin", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.AnalyzeCoin(ctx, AnalyzeProxyRequest{LLM: llm})
			return err
		}},
		{"GradeCoin", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.GradeCoin(ctx, GradeProxyRequest{LLM: llm})
			return err
		}},
		{"GenerateIntakeDraft", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.GenerateIntakeDraft(ctx, llm, []string{"img"}, nil)
			return err
		}},
		{"CheckAvailability", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.CheckAvailability(ctx, AvailabilityCheckProxyRequest{LLM: llm})
			return err
		}},
		{"GetBidMarketSignal", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.GetBidMarketSignal(ctx, BidMarketSignalProxyRequest{LLM: llm})
			return err
		}},
		{"RunSetBuilder", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.RunSetBuilder(ctx, SetBuilderProxyRequest{LLM: llm})
			return err
		}},
		{"DiscoverAlertCandidates", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.DiscoverAlertCandidates(ctx, AlertDiscoveryProxyRequest{LLM: llm})
			return err
		}},
		{"SearchComparables", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.SearchComparables(ctx, ComparablesProxyRequest{LLM: llm})
			return err
		}},
		{"WishlistFeaturedSummary", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.WishlistFeaturedSummary(ctx, WishlistFeaturedSummaryProxyRequest{LLM: llm})
			return err
		}},
		{"ExtractWishlistURL", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.ExtractWishlistURL(ctx, WishlistURLExtractionRequest{LLM: llm, SourceURL: "https://dealer.example/lot/1"})
			return err
		}},
		{"CollectPortfolioReview", func(ctx context.Context, p *AgentProxy) error {
			_, err := p.CollectPortfolioReview(ctx, PortfolioReviewProxyRequest{LLM: llm})
			return err
		}},
		{"StreamChat", func(ctx context.Context, p *AgentProxy) error {
			return p.StreamChat(ctx, httptest.NewRecorder(), AgentChatProxyRequest{LLM: llm})
		}},
		{"StreamDeepIdentification", func(ctx context.Context, p *AgentProxy) error {
			return p.StreamDeepIdentification(ctx, DeepIdentifyProxyRequest{LLM: llm}, func(DeepIdentifyFrame) error { return nil })
		}},
	}
}

func TestAgentProxyNon200ErrorBodiesDoNotLeakSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(agentErrorBodyWithSecret))
	}))
	defer server.Close()

	for _, tc := range agentProxyCalls() {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewLogger(50)
			proxy := NewAgentProxy(server.URL, "internal-token", logger)
			err := tc.call(context.Background(), proxy)
			if err == nil {
				t.Fatal("expected an error for a non-200 agent response")
			}
			if strings.Contains(err.Error(), leakedProviderKey) {
				t.Fatalf("returned error leaked provider key: %v", err)
			}
			for _, entry := range logger.GetLogs(50) {
				if strings.Contains(entry.Message, leakedProviderKey) {
					t.Fatalf("log entry leaked provider key: %s", entry.Message)
				}
			}
		})
	}
}

// Non-JSON bodies are truncated, not redacted, in logs; returned errors must never echo them.
func TestAgentProxyReturnedErrorsNeverEchoPlainTextBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("provider rejected key " + leakedProviderKey))
	}))
	defer server.Close()

	for _, tc := range agentProxyCalls() {
		t.Run(tc.name, func(t *testing.T) {
			proxy := NewAgentProxy(server.URL, "internal-token", NewLogger(50))
			err := tc.call(context.Background(), proxy)
			if err == nil {
				t.Fatal("expected an error for a non-200 agent response")
			}
			if strings.Contains(err.Error(), leakedProviderKey) {
				t.Fatalf("returned error echoed the agent body: %v", err)
			}
		})
	}
}

func TestAgentProxyCallerTimeoutCancelsAgentRequest(t *testing.T) {
	for _, tc := range agentProxyCalls() {
		t.Run(tc.name, func(t *testing.T) {
			cancelled := make(chan struct{}, 1)
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// net/http only watches for a client disconnect once the body is consumed.
				_, _ = io.Copy(io.Discard, r.Body)
				select {
				case <-r.Context().Done():
					cancelled <- struct{}{}
				case <-release:
				case <-time.After(3 * time.Second):
				}
			}))
			defer server.Close()
			defer close(release)

			proxy := NewAgentProxy(server.URL, "internal-token", NewLogger(10))
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := tc.call(ctx, proxy); err == nil {
				t.Fatal("expected an error when the caller context times out")
			}
			select {
			case <-cancelled:
			case <-time.After(5 * time.Second):
				t.Fatal("agent request was not cancelled when the caller context timed out")
			}
		})
	}
}
