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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kazhuravlev/database-gateway/internal/policy"
	oparego "github.com/open-policy-agent/opa/v1/rego"
)

const (
	queryAllowTarget = "x = data.gateway.allow_target"
	queryAllowQuery  = "x = data.gateway.allow_query"
)

type Authorizer struct {
	targetQuery oparego.PreparedEvalQuery
	queryQuery  oparego.PreparedEvalQuery
}

var _ policy.Authorizer = (*Authorizer)(nil)

func New(ctx context.Context, modules map[string]string) (*Authorizer, error) {
	targetQuery, err := prepareQuery(ctx, modules, queryAllowTarget)
	if err != nil {
		return nil, fmt.Errorf("prepare target query: %w", err)
	}

	queryQuery, err := prepareQuery(ctx, modules, queryAllowQuery)
	if err != nil {
		return nil, fmt.Errorf("prepare vector query: %w", err)
	}

	return &Authorizer{
		targetQuery: targetQuery,
		queryQuery:  queryQuery,
	}, nil
}

func (a *Authorizer) AllowTarget(subjects []string, target string) bool {
	return evalBool(a.targetQuery, policyInput{
		Subjects: subjects,
		Target:   target,
		Op:       "",
		Table:    "",
	})
}

func (a *Authorizer) AllowQuery(subjects []string, target, op, table string) bool {
	return evalBool(a.queryQuery, policyInput{
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

func LoadModules(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read policy dir: %w", err)
	}

	modules := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".rego" {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		buf, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read policy module %q: %w", filename, err)
		}

		modules[entry.Name()] = string(buf)
	}

	if len(modules) == 0 {
		return nil, fmt.Errorf("no .rego files found in %q", dir) //nolint:err113
	}

	return modules, nil
}

func SubjectUser(userID string) string {
	return "user:" + strings.TrimSpace(userID)
}

func SubjectRole(role string) string {
	return "role:" + strings.TrimSpace(role)
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
