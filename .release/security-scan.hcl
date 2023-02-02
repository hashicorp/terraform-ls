binary {
	go_modules = true # Scan the Go modules found in the binary
	osv        = true  # Use the Open Source Vulnerabilities (OSV) database
	oss_index  = true  # Use the Sonatype OSS Index vulnerability database
	nvd        = true  # Use the Nation Vulnerability Database

	secrets { # Scan for secrets in the binary
		all = true 
	}
}