# Copyright IBM Corp. 2020, 2026
# SPDX-License-Identifier: MPL-2.0

binary {
	go_modules = true # Scan the Go modules found in the binary
	osv        = true  # Use the Open Source Vulnerabilities (OSV) database
	oss_index  = true  # Use the Sonatype OSS Index vulnerability database
	nvd        = true  # Use the Nation Vulnerability Database

	secrets { # Scan for secrets in the binary
		all = true 
	}

	triage {
		suppress {
			vulnerabilities = [
				# suppressed temporarily as per: 
				# https://github.com/hashicorp/terraform/pull/38332/changes
				# https://github.com/hashicorp/terraform-provider-aws/pull/48838/changes
				"GO-2026-5932",
			]
		}
	}
}