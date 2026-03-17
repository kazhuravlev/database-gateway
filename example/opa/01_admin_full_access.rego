package gateway

default allow_target := false
default allow_vector := false

# Platform administrators can see every target and run every operation.
allow_target if {
	"role:admin" in input.subjects
}

allow_vector if {
	"role:admin" in input.subjects
}

