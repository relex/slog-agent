// Package fluentdforward provides output implementations for fluentd "Forward" protocol, split into:
//
// - eventSerializer serializes log records into msgpack formatted events one by one
//
// - messagePacker joins and compresses events into msgpack Forward messages, equals to entire requests in the protocol.
//
// - clientWorker sends out the messages to upstream fluentd, and handles other protocol parts such as auth and ping.
package fluentdforward
