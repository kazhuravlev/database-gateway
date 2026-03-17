package gateway

default allow_target := false
default allow_vector := false

# Finance has access to payment data in production, with a tighter table set.
finance_tables := {
	"public.transactions",
	"public.transfers",
}

allow_target if {
	"role:finance" in input.subjects
	input.target == "taxi-prod"
}

allow_vector if {
	"role:finance" in input.subjects
	input.target == "taxi-prod"
	input.op == "select"
	input.table in finance_tables
}

allow_vector if {
	"role:finance-manager" in input.subjects
	input.target == "taxi-prod"
	input.op == "update"
	input.table == "public.transactions"
}

