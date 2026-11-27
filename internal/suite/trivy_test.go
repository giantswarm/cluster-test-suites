package suite

import "testing"

func TestTrivyOperatorDependsOnValues(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{version: "2.4.0", want: trivyOperatorDependsOnList},
		{version: "v2.4.9", want: trivyOperatorDependsOnList},
		{version: "2.5.0", want: trivyOperatorDependsOnMap},
		{version: "2.5.0-rc.1", want: trivyOperatorDependsOnMap},
		{version: "2.5.0-0b21ef56638dbd9c4d3e3a0e2f1c8a7b6d5e4f3a2", want: trivyOperatorDependsOnMap},
		{version: "3.0.0", want: trivyOperatorDependsOnMap},
		{version: "latest", wantErr: true},
		{version: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			got, err := trivyOperatorDependsOnValues(tc.version)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, want error: %t", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestEnablesTrivyOperator(t *testing.T) {
	tests := []struct {
		name   string
		values string
		want   bool
	}{
		{
			name: "enabled",
			values: `global:
  apps:
    securityBundle:
      values:
        apps:
          trivyOperator:
            enabled: true
`,
			want: true,
		},
		{
			name: "disabled",
			values: `global:
  apps:
    securityBundle:
      values:
        apps:
          trivyOperator:
            enabled: false
`,
		},
		{name: "not mentioned", values: "global:\n  metadata:\n    name: test\n"},
		{name: "empty", values: ""},
		{name: "templated values that are not valid YAML", values: "global:\n  {{ .ClusterName }\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := enablesTrivyOperator(tc.values); got != tc.want {
				t.Errorf("got %t, want %t", got, tc.want)
			}
		})
	}
}
