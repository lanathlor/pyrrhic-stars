class_name ServerConfig
extends RefCounted

## Resolves backend endpoint URLs from the configured server address.
##
## Local/dev runs serve Kratos, the gateway WebSocket and the settings REST API
## as plaintext on a single host at distinct ports (docker-compose). Production
## fronts them with a TLS ingress on split hostnames (auth.<domain>, api.<domain>)
## on 443. An IP literal or "localhost" -- or SERVER_INSECURE=1 -- selects the dev
## shape; a DNS hostname selects the secure shape.

const KRATOS_PORT := 4433
const GATEWAY_PORT := 7777


## True when the host should be reached over plaintext at its dev ports rather
## than the production TLS ingress.
static func is_insecure(host: String) -> bool:
	if OS.get_environment("SERVER_INSECURE") == "1":
		return true
	if host == "localhost" or host == "":
		return true
	return host.is_valid_ip_address()


## Kratos self-service base, e.g. https://auth.example.com or http://127.0.0.1:4433
static func auth_base(host: String) -> String:
	if is_insecure(host):
		return "http://%s:%d" % [host, KRATOS_PORT]
	return "https://auth.%s" % host


## Gateway HTTP base for the settings REST API.
static func gateway_http_base(host: String) -> String:
	if is_insecure(host):
		return "http://%s:%d" % [host, GATEWAY_PORT]
	return "https://api.%s" % host


## Gateway WebSocket base.
static func gateway_ws_base(host: String) -> String:
	if is_insecure(host):
		return "ws://%s:%d" % [host, GATEWAY_PORT]
	return "wss://api.%s" % host
