package sam

import (
	"github.com/liamg/iamgo"

	"github.com/aquasecurity/defsec/pkg/providers/aws/iam"
	"github.com/aquasecurity/defsec/pkg/providers/aws/sam"
	defsecTypes "github.com/aquasecurity/defsec/pkg/types"
	parser2 "github.com/aquasecurity/trivy/pkg/iac/scanners/cloudformation/parser"
)

func getFunctions(cfFile parser2.FileContext) (functions []sam.Function) {

	functionResources := cfFile.GetResourcesByType("AWS::Serverless::Function")
	for _, r := range functionResources {
		function := sam.Function{
			Metadata:        r.Metadata(),
			FunctionName:    r.GetStringProperty("FunctionName"),
			Tracing:         r.GetStringProperty("Tracing", sam.TracingModePassThrough),
			ManagedPolicies: nil,
			Policies:        nil,
		}

		setFunctionPolicies(r, &function)
		functions = append(functions, function)
	}

	return functions
}

func setFunctionPolicies(r *parser2.Resource, function *sam.Function) {
	policies := r.GetProperty("Policies")
	if policies.IsNotNil() {
		if policies.IsString() {
			function.ManagedPolicies = append(function.ManagedPolicies, policies.AsStringValue())
		} else if policies.IsList() {
			for _, property := range policies.AsList() {
				if property.IsMap() {
					parsed, err := iamgo.Parse(property.GetJsonBytes(true))
					if err != nil {
						continue
					}
					policy := iam.Policy{
						Metadata: property.Metadata(),
						Name:     defsecTypes.StringDefault("", property.Metadata()),
						Document: iam.Document{
							Metadata: property.Metadata(),
							Parsed:   *parsed,
						},
						Builtin: defsecTypes.Bool(false, property.Metadata()),
					}
					function.Policies = append(function.Policies, policy)
				} else if property.IsString() {
					function.ManagedPolicies = append(function.ManagedPolicies, property.AsStringValue())
				}
			}
		}
	}
}
