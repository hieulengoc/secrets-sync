package utils

import (
	"reflect"
	"testing"
)

func TestGetConfigFromFile(t *testing.T) {
	type args struct {
		fName string
	}
	tests := []struct {
		name    string
		args    args
		want    *secretList
		wantErr bool
	}{
		// TODO: Add test cases.
		struct {
			name    string
			args    args
			want    *secretList
			wantErr bool
		}{
			name: "test read file",
			args: args{
				fName: "test_secrets.yaml",
			},
			want: &secretList{
				Secrets: []Secret{
					Secret{
						Name:            "secret1",
						SourceNamespace: "source_namespace1",
						TargetNamespaces: []string{
							"target_namespace1",
						},
					},
					Secret{
						Name:            "secret2",
						SourceNamespace: "source_namespace2",
						TargetNamespaces: []string{
							"target_namespace2",
							"target_namespace3",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfigFromFile(tt.args.fName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfigFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
