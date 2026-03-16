// Database Gateway provides access to servers with ACL for safe and restricted database interactions.
// Copyright (C) 2024  Kirill Zhuravlev
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package opa

import (
	"context"
	"fmt"
	"slices"

	"github.com/kazhuravlev/database-gateway/internal/policy"
	oparego "github.com/open-policy-agent/opa/v1/rego"
)

const (
	queryAllowTarget = "x = data.gateway.allow_target"
	queryAllowVector = "x = data.gateway.allow_vector"
)

type Authorizer struct {
	targetQuery oparego.PreparedEvalQuery
	vectorQuery oparego.PreparedEvalQuery
}

var _ policy.Authorizer = (*Authorizer)(nil)

func New(ctx context.Context, modules map[string]string) (*Authorizer, error) {
	targetQuery, err := prepareQuery(ctx, modules, queryAllowTarget)
	if err != nil {
		return nil, fmt.Errorf("prepare target query: %w", err)
	}

	vectorQuery, err := prepareQuery(ctx, modules, queryAllowVector)
	if err != nil {
		return nil, fmt.Errorf("prepare vector query: %w", err)
	}

	return &Authorizer{
		targetQuery: targetQuery,
		vectorQuery: vectorQuery,
	}, nil
}

func (a *Authorizer) AllowTarget(subjects []string, target string) bool {
	return evalBool(a.targetQuery, policyInput{
		Subjects: subjects,
		Target:   target,
	})
}

func (a *Authorizer) AllowVector(subjects []string, target, op, table string) bool {
	return evalBool(a.vectorQuery, policyInput{
		Subjects: subjects,
		Target:   target,
		Op:       op,
		Table:    table,
	})
}

type policyInput struct {
	Subjects []string `json:"subjects"`
	Target   string   `json:"target"`
	Op       string   `json:"op,omitempty"`
	Table    string   `json:"table,omitempty"`
}

func prepareQuery(
	ctx context.Context,
	modules map[string]string,
	query string,
) (oparego.PreparedEvalQuery, error) {
	moduleNames := make([]string, 0, len(modules))
	for name := range modules {
		moduleNames = append(moduleNames, name)
	}
	slices.Sort(moduleNames)

	opts := make([]func(*oparego.Rego), 0, len(moduleNames)+1)
	opts = append(opts, oparego.Query(query))
	for _, name := range moduleNames {
		opts = append(opts, oparego.Module(name, modules[name]))
	}

	return oparego.New(opts...).PrepareForEval(ctx)
}

func evalBool(query oparego.PreparedEvalQuery, input policyInput) bool {
	results, err := query.Eval(context.Background(), oparego.EvalInput(input))
	if err != nil || len(results) == 0 {
		return false
	}

	value, ok := results[0].Bindings["x"].(bool)
	if !ok {
		return false
	}

	return value
}
