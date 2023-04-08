package model

//go:generate go run github.com/hannahhoward/cbor-gen-for --map-encoding QueryResponse
type QueryResponse struct {
	Protocols []Protocol
}
