// integrations.go provides data-driven classification of processes as known
// infrastructure integrations (Redis, MySQL, Kafka, etc.) based on listening
// ports, executable names, and command-line patterns.
//
// Adding a new integration: append an integrationRule to the integrationRules
// slice. No new files, interfaces, or registry entries needed.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// IntegrationType identifies a known infrastructure integration.
type IntegrationType string

const (
	IntegrationRedis         IntegrationType = "redis"
	IntegrationMySQL         IntegrationType = "mysql"
	IntegrationPostgreSQL    IntegrationType = "postgresql"
	IntegrationMongoDB       IntegrationType = "mongodb"
	IntegrationKafka         IntegrationType = "kafka"
	IntegrationRabbitMQ      IntegrationType = "rabbitmq"
	IntegrationElasticsearch IntegrationType = "elasticsearch"
	IntegrationCassandra     IntegrationType = "cassandra"
	IntegrationMemcached     IntegrationType = "memcached"
	IntegrationNginx         IntegrationType = "nginx"
	IntegrationApache        IntegrationType = "apache"
	IntegrationHAProxy       IntegrationType = "haproxy"
	IntegrationZookeeper     IntegrationType = "zookeeper"
	IntegrationEtcd          IntegrationType = "etcd"
	IntegrationConsul        IntegrationType = "consul"
	IntegrationVault         IntegrationType = "vault"
	IntegrationCouchDB       IntegrationType = "couchdb"
	IntegrationClickHouse    IntegrationType = "clickhouse"
	IntegrationInfluxDB      IntegrationType = "influxdb"
	IntegrationNATS          IntegrationType = "nats"
)

// MatchStrategy controls how port and process signals combine in a rule.
type MatchStrategy int

const (
	// MatchOr means either port OR process name is sufficient.
	// Use for unique ports (6379 = almost certainly Redis).
	MatchOr MatchStrategy = iota

	// MatchAnd means both port AND process name are required.
	// Use for shared ports (80/8080 could be anything).
	MatchAnd

	// MatchProcessRequired means process name must match; port is optional.
	// Use for services with distinctive exe names that may listen on any port
	// (e.g. nginx on port 81, apache2 on port 8443). A port-only match is
	// never sufficient — the exe name (or cmdline) is the real signal.
	MatchProcessRequired
)

// integrationRule defines a single classification rule.
type integrationRule struct {
	Type     IntegrationType
	Strategy MatchStrategy

	// Ports that this integration typically listens on.
	Ports []uint16

	// ExeNames matched exactly against Process.ExecutableName (case-insensitive).
	ExeNames []string

	// ExeContains matched as substrings against Process.ExecutableName (case-insensitive).
	ExeContains []string

	// CmdContains matched as substrings against Process.CommandLine (case-insensitive).
	CmdContains []string

	// ExcludeExe: if ExecutableName contains any of these (case-insensitive),
	// the rule is skipped. Filters out client tools like redis-cli, psql, etc.
	ExcludeExe []string

	// ExcludeCmd: if CommandLine contains any of these (case-insensitive),
	// the rule is skipped. Filters out non-server processes like walwriter.
	ExcludeCmd []string
}

// integrationRules is the classification rule table. Rules are evaluated in
// order; first match wins. More-specific rules must come before less-specific
// ones (e.g., RabbitMQ before CouchDB since both use beam.smp).
var integrationRules = []integrationRule{
	{
		Type:        IntegrationRedis,
		Strategy:    MatchOr,
		Ports:       []uint16{6379},
		ExeNames:    []string{"redis-server", "valkey-server", "keydb-server"},
		ExcludeExe:  []string{"redis-cli", "redis-benchmark", "redis-check-aof", "redis-check-rdb", "redis-sentinel"},
	},
	{
		Type:        IntegrationMySQL,
		Strategy:    MatchOr,
		Ports:       []uint16{3306},
		ExeNames:    []string{"mysqld", "mariadbd"},
		ExeContains: []string{"mysqld", "mariadbd"},
		ExcludeExe:  []string{"mysql-client", "mysqldump", "mysqladmin", "mysqlcheck", "mysqlimport", "mysqlbinlog"},
	},
	{
		Type:        IntegrationPostgreSQL,
		Strategy:    MatchOr,
		Ports:       []uint16{5432},
		ExeNames:    []string{"postgres"},
		ExcludeExe:  []string{"pg_dump", "pg_restore", "psql", "pg_basebackup", "pg_isready"},
		ExcludeCmd:  []string{"logger", "checkpointer", "background writer", "walwriter", "autovacuum launcher", "stats collector", "logical replication"},
	},
	{
		Type:        IntegrationMongoDB,
		Strategy:    MatchOr,
		Ports:       []uint16{27017},
		ExeNames:    []string{"mongod", "mongos"},
		ExcludeExe:  []string{"mongo", "mongosh", "mongodump", "mongorestore", "mongoexport", "mongoimport", "mongostat", "mongotop"},
	},
	{
		Type:        IntegrationKafka,
		Strategy:    MatchOr,
		Ports:       []uint16{9092},
		CmdContains: []string{"kafka.kafka", "kafka.server.kafkaserver", "io.confluent.kafka"},
		ExcludeCmd:  []string{"kafka-topics", "kafka-console-producer", "kafka-console-consumer", "kafka-configs", "kafka-consumer-groups", "kafka.tools."},
	},
	{
		// RabbitMQ runs on Erlang VM (beam.smp). Must come before CouchDB.
		Type:        IntegrationRabbitMQ,
		Strategy:    MatchOr,
		Ports:       []uint16{5672},
		CmdContains: []string{"rabbit"},
		ExcludeExe:  []string{"rabbitmqctl", "rabbitmqadmin", "rabbitmq-plugins", "rabbitmq-diagnostics"},
	},
	{
		Type:        IntegrationElasticsearch,
		Strategy:    MatchOr,
		Ports:       []uint16{9200},
		CmdContains: []string{"org.elasticsearch.bootstrap.elasticsearch", "opensearch"},
		ExcludeCmd:  []string{"elasticsearch-setup-passwords", "elasticsearch-keystore"},
	},
	{
		Type:        IntegrationCassandra,
		Strategy:    MatchOr,
		Ports:       []uint16{9042},
		CmdContains: []string{"org.apache.cassandra.service.cassandradaemon", "cassandradaemon"},
		ExcludeExe:  []string{"cqlsh", "nodetool", "cassandra-stress"},
	},
	{
		Type:     IntegrationMemcached,
		Strategy: MatchOr,
		Ports:    []uint16{11211},
		ExeNames: []string{"memcached"},
	},
	{
		Type:        IntegrationZookeeper,
		Strategy:    MatchOr,
		Ports:       []uint16{2181},
		CmdContains: []string{"org.apache.zookeeper.server"},
	},
	{
		Type:     IntegrationEtcd,
		Strategy: MatchOr,
		Ports:    []uint16{2379},
		ExeNames: []string{"etcd"},
	},
	{
		// Consul has a distinctive exe name; port 8500 alone is too ambiguous.
		Type:        IntegrationConsul,
		Strategy:    MatchProcessRequired,
		ExeNames:    []string{"consul"},
		CmdContains: []string{"consul agent"},
	},
	{
		// Vault has a distinctive exe name; port 8200 alone is too ambiguous.
		Type:        IntegrationVault,
		Strategy:    MatchProcessRequired,
		ExeNames:    []string{"vault"},
		CmdContains: []string{"vault server"},
	},
	{
		// CouchDB also runs on Erlang VM (beam.smp). Must come after RabbitMQ.
		Type:        IntegrationCouchDB,
		Strategy:    MatchOr,
		Ports:       []uint16{5984},
		CmdContains: []string{"couchdb"},
		ExcludeCmd:  []string{"rabbit"},
	},
	{
		// ClickHouse has distinctive exe names; ports 8123/9000 are ambiguous.
		Type:        IntegrationClickHouse,
		Strategy:    MatchProcessRequired,
		ExeNames:    []string{"clickhouse-server", "clickhouse"},
		ExcludeExe:  []string{"clickhouse-client", "clickhouse-local", "clickhouse-benchmark"},
	},
	{
		// InfluxDB has a distinctive exe name; port 8086 is ambiguous.
		Type:     IntegrationInfluxDB,
		Strategy: MatchProcessRequired,
		ExeNames: []string{"influxd"},
	},
	{
		Type:     IntegrationNATS,
		Strategy: MatchOr,
		Ports:    []uint16{4222},
		ExeNames: []string{"nats-server"},
	},
	{
		// Nginx/Apache/HAProxy have distinctive exe names but listen on any
		// port — process name alone is the signal, port is not required.
		Type:     IntegrationNginx,
		Strategy: MatchProcessRequired,
		ExeNames: []string{"nginx"},
	},
	{
		Type:     IntegrationApache,
		Strategy: MatchProcessRequired,
		ExeNames: []string{"httpd", "apache2"},
	},
	{
		Type:     IntegrationHAProxy,
		Strategy: MatchProcessRequired,
		ExeNames: []string{"haproxy"},
	},
}

// globalExcludeExe is a set of executable names that should never match any
// integration rule. docker-proxy is Docker's userland port forwarder — it
// can listen on any port and would false-positive on port-based rules.
var globalExcludeExe = map[string]struct{}{
	"docker-proxy": {},
}

// ClassifyIntegration checks a Process against the integration rule table
// and returns the first matching integration type, or "" if no rule matches.
// Requires Process.Listeners() to be populated (call after AttachListeners).
func ClassifyIntegration(proc *Process) IntegrationType {
	exeLower := strings.ToLower(proc.ExecutableName)
	cmdLower := strings.ToLower(proc.CommandLine)

	if _, excluded := globalExcludeExe[exeLower]; excluded {
		return ""
	}

	listeners := proc.Listeners()
	portSet := make(map[uint16]struct{}, len(listeners))
	for _, l := range listeners {
		portSet[l.Port] = struct{}{}
	}

	for i := range integrationRules {
		rule := &integrationRules[i]

		if isExcluded(exeLower, cmdLower, rule) {
			continue
		}

		portMatch := matchPort(portSet, rule.Ports)
		procMatch := matchProcess(exeLower, cmdLower, rule)

		switch rule.Strategy {
		case MatchOr:
			if portMatch || procMatch {
				return rule.Type
			}
		case MatchAnd:
			if portMatch && procMatch {
				return rule.Type
			}
		case MatchProcessRequired:
			if procMatch {
				return rule.Type
			}
		}
	}

	return ""
}

// classifyIntegrations runs integration classification on a batch of
// enriched Processes. Called after AttachListeners() in the pipeline.
func classifyIntegrations(procs []*Process) {
	for _, proc := range procs {
		if itype := ClassifyIntegration(proc); itype != "" {
			if proc.Details == nil {
				proc.Details = make(map[string]any)
			}
			proc.Details[DetailIntegrationType] = string(itype)
		}
	}
}

func isExcluded(exeLower, cmdLower string, rule *integrationRule) bool {
	for _, excl := range rule.ExcludeExe {
		if exeLower == strings.ToLower(excl) {
			return true
		}
	}
	for _, excl := range rule.ExcludeCmd {
		if strings.Contains(cmdLower, strings.ToLower(excl)) {
			return true
		}
	}
	return false
}

func matchPort(portSet map[uint16]struct{}, rulePorts []uint16) bool {
	for _, p := range rulePorts {
		if _, ok := portSet[p]; ok {
			return true
		}
	}
	return false
}

func matchProcess(exeLower, cmdLower string, rule *integrationRule) bool {
	for _, name := range rule.ExeNames {
		if exeLower == strings.ToLower(name) {
			return true
		}
	}
	for _, sub := range rule.ExeContains {
		if strings.Contains(exeLower, strings.ToLower(sub)) {
			return true
		}
	}
	for _, sub := range rule.CmdContains {
		if strings.Contains(cmdLower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// DiscoverIntegrations scans all processes against integration rules and
// returns Process structs for matched integrations. Processes already claimed
// by a language handler (passed in skipPIDs) are excluded — they get
// IntegrationType annotated via classifyIntegrations in the language pipeline.
func (d *discoverer) DiscoverIntegrations(ctx context.Context, skipPIDs map[int32]struct{}) ([]*Process, error) {
	start := time.Now()
	log := d.logger()

	var candidates []*Process
	for i := range d.allProcesses {
		info := &d.allProcesses[i]
		if _, skip := skipPIDs[info.PID]; skip {
			continue
		}

		proc := &Process{
			PID:            info.PID,
			ParentPID:      readProcessPPID(info.PID),
			ExecutableName: info.ExeName,
			ExecutablePath: info.ExePath,
			Command:        info.CmdLine,
			CommandLine:    info.CmdLine,
			CommandArgs:    info.CmdArgs,
			Owner:          readProcessOwner(info.PID),
			CreateTime:     timeFromMillis(readProcessCreateTime(info.PID)),
			Status:         readProcessStatus(info.PID),
			Details:        make(map[string]any),
		}

		if d.opts.IncludeContainerInfo || d.opts.ExcludeContainers {
			containerInfo, err := d.containerDetector.IsProcessInContainer(info.PID)
			if err == nil {
				proc.ContainerInfo = containerInfo
				if d.opts.ExcludeContainers && containerInfo.IsContainer {
					continue
				}
			}
		}

		enrichCommonDetails(proc)
		candidates = append(candidates, proc)
	}

	AttachListeners(candidates)
	InheritParentPorts(candidates)

	var matched []*Process
	for _, proc := range candidates {
		if itype := ClassifyIntegration(proc); itype != "" {
			proc.Details[DetailIntegrationType] = string(itype)
			proc.ServiceName = proc.ExecutableName
			matched = append(matched, proc)
		}
	}

	batchResolveContainerNames(ctx, matched, d.containerClients)
	applyContainerServiceNames(matched)

	log.Debug("integration discovery complete",
		"scanned_pids", len(d.allProcesses),
		"skipped_pids", len(skipPIDs),
		"candidates", len(candidates),
		"matched", len(matched),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return matched, nil
}

// FindIntegrations discovers infrastructure integrations (Redis, MySQL, etc.)
// running on the host. Language-classified processes are excluded — they get
// IntegrationType annotated in the language pipeline instead.
func FindIntegrations(ctx context.Context, logger *slog.Logger) ([]*Process, error) {
	opts := DefaultDiscoveryOptions()
	opts.ExcludeContainers = false
	opts.IncludeContainerInfo = true
	opts.Logger = logger

	d, err := NewDiscovererWithOptions(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error discovering integrations: %w", err)
	}
	defer d.Close()

	grouped := d.classifyAll()
	skipPIDs := make(map[int32]struct{})
	for _, infos := range grouped {
		for _, info := range infos {
			skipPIDs[info.PID] = struct{}{}
		}
	}

	return d.DiscoverIntegrations(ctx, skipPIDs)
}

// FindAllWithIntegrations discovers both language-classified processes and
// infrastructure integrations in a single scan. Returns the language-grouped
// processes and integration processes separately.
func FindAllWithIntegrations(ctx context.Context, logger *slog.Logger) (map[Language][]*Process, []*Process, error) {
	opts := DefaultDiscoveryOptions()
	opts.ExcludeContainers = false
	opts.IncludeContainerInfo = true
	opts.Logger = logger

	d, err := NewDiscovererWithOptions(ctx, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("error discovering processes: %w", err)
	}
	defer d.Close()

	langResult, err := d.DiscoverAll(ctx)
	if err != nil {
		return nil, nil, err
	}

	skipPIDs := make(map[int32]struct{})
	for _, procs := range langResult {
		for _, proc := range procs {
			skipPIDs[proc.PID] = struct{}{}
		}
	}

	integrations, err := d.DiscoverIntegrations(ctx, skipPIDs)
	if err != nil {
		return langResult, nil, err
	}

	return langResult, integrations, nil
}
