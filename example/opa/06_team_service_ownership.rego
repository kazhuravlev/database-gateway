package gateway

default allow_target := false
default allow_vector := false

# Team ownership encoded in target ids:
# payments-prod, payments-staging, search-dev, etc.
allow_target if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-")
}

allow_vector if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-dev")
}

allow_vector if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-staging")
	input.op == "select"
}

allow_vector if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-prod")
	input.op == "select"
}

