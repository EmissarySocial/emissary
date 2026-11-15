package consumer

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/floodgate"
	"github.com/benpate/remote"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// TaskProbeDormantHosts is the daily task that checks whether dormant hosts have come back
const TaskProbeDormantHosts = "ProbeDormantHosts"

// probeTimeout bounds the whole task, so that a list of dead hosts cannot hold a worker for hours
const probeTimeout = 10 * time.Minute

// probeMaxResponseSize is the largest NodeInfo answer that a probe reads
const probeMaxResponseSize = 1 << 16

// hostProber checks whether one remote host is running, and what software it runs
type hostProber interface {
	probe(host string) (software string, version string, err error)
}

// ProbeDormantHosts checks each host that Floodgate has marked dormant and that is due for
// a check, and revives the ones that answer
func ProbeDormantHosts(serverFactory ServerFactory) queue.Result {

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	prober := nodeInfoProber{allowPrivateIPs: serverFactory.AllowPrivateIPs()}

	return probeDormantHosts(ctx, serverFactory.Floodgate(), prober)
}

// probeDormantHosts runs one check against every host that is due, and records each outcome
func probeDormantHosts(ctx context.Context, gate floodgate.Gate, prober hostProber) queue.Result {

	const location = "consumer.probeDormantHosts"

	hosts, err := gate.ProbesDue(ctx)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Loading hosts to check"))
	}

	for host := range hosts {

		// RULE: Stop when the task's time is up. The hosts that were not reached stay due, and
		// tomorrow's task checks them.
		if ctx.Err() != nil {
			break
		}

		if err := probeHost(ctx, gate, prober, host); err != nil {
			derp.Report(derp.Wrap(err, location, "Recording check", "host: "+host.Host))
		}
	}

	// Anybody home?
	return queue.Success()
}

// probeHost checks one host, then revives it or schedules its next check
func probeHost(ctx context.Context, gate floodgate.Gate, prober hostProber, host floodgate.Host) error {

	software, version, err := prober.probe(host.Host)

	// A host that does not answer is the expected case, so it is logged and not reported
	if err != nil {
		log.Debug().Str("host", host.Host).Err(err).Msg("Dormant host did not answer")
		return gate.ProbeFailed(ctx, host)
	}

	return gate.ProbeSucceeded(ctx, host.Host, software, version)
}

// nodeInfoProber checks a host by fetching its NodeInfo discovery document, without a signature
type nodeInfoProber struct {
	allowPrivateIPs bool
}

// nodeInfoDiscovery is the document served at /.well-known/nodeinfo
type nodeInfoDiscovery struct {
	Links []struct {
		Href string `json:"href"`
	} `json:"links"`
}

// nodeInfoSoftware is the part of a NodeInfo document that names the server software
type nodeInfoSoftware struct {
	Software struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"software"`
}

// probe returns the software that a host reports. An error means that the host did not answer;
// a host that answers without naming its software is running, and returns empty strings.
func (prober nodeInfoProber) probe(host string) (string, string, error) {

	const location = "consumer.nodeInfoProber.probe"

	// The host is running if it serves its discovery document
	discovery := nodeInfoDiscovery{}

	if err := prober.get("https://"+host+"/.well-known/nodeinfo", &discovery); err != nil {
		return "", "", derp.Wrap(err, location, "Loading NodeInfo discovery document", "host: "+host)
	}

	// The software name is a courtesy. Failing to read it does not count against the host.
	documentURL := nodeInfoDocumentURL(host, discovery)

	if documentURL == "" {
		return "", "", nil
	}

	document := nodeInfoSoftware{}

	if err := prober.get(documentURL, &document); err != nil {
		return "", "", nil
	}

	return document.Software.Name, document.Software.Version, nil
}

// get fetches one small JSON document, under the server's rule for private addresses
func (prober nodeInfoProber) get(documentURL string, result any) error {

	return remote.Get(documentURL).
		Accept("application/json").
		AllowPrivateIPs(prober.allowPrivateIPs).
		MaxResponseSize(probeMaxResponseSize).
		Result(result).
		Send()
}

// nodeInfoDocumentURL returns the last NodeInfo document that a discovery document links to on
// the host itself, or an empty string when it links to none
func nodeInfoDocumentURL(host string, discovery nodeInfoDiscovery) string {

	result := ""

	for _, link := range discovery.Links {

		parsed, err := url.Parse(link.Href)

		if err != nil {
			continue
		}

		// RULE: Never follow a link to another host. A hostile server could aim this server's
		// requests at a third party.
		if (parsed.Scheme != "https") || !strings.EqualFold(parsed.Host, host) {
			continue
		}

		result = link.Href
	}

	return result
}
