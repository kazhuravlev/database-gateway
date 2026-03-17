package gateway

default allow_target := false
default allow_query := false

# Support can look up operational records, but not highly sensitive tables.
allowed_support_tables := {
	"public.clients",
	"public.transfers",
	"public.transactions",
}

allow_target if {
	"role:support" in input.subjects
	input.target == "taxi-prod"
}

allow_query if {
	"role:support" in input.subjects
	input.target == "taxi-prod"
	input.op == "select"
	input.table in allowed_support_tables
}

