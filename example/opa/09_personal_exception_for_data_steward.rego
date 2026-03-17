package gateway

default allow_target := false
default allow_query := false

# Data stewards get broad read access plus named write access for remediation.
allow_target if {
	"role:data-steward" in input.subjects
	input.target == "taxi-prod"
}

allow_query if {
	"role:data-steward" in input.subjects
	input.target == "taxi-prod"
	input.op == "select"
}

allow_query if {
	"user:steward.lead@example.com" in input.subjects
	input.target == "taxi-prod"
	input.op == "update"
	input.table == "public.clients"
}
