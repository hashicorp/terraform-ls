schema = 1
artifacts {
  zip = [
    "terraform-ls_${version}_darwin_amd64.zip",
    "terraform-ls_${version}_darwin_arm64.zip",
    "terraform-ls_${version}_freebsd_386.zip",
    "terraform-ls_${version}_freebsd_amd64.zip",
    "terraform-ls_${version}_freebsd_arm.zip",
    "terraform-ls_${version}_linux_386.zip",
    "terraform-ls_${version}_linux_amd64.zip",
    "terraform-ls_${version}_linux_arm.zip",
    "terraform-ls_${version}_linux_arm64.zip",
    "terraform-ls_${version}_openbsd_386.zip",
    "terraform-ls_${version}_openbsd_amd64.zip",
    "terraform-ls_${version}_solaris_amd64.zip",
    "terraform-ls_${version}_windows_386.zip",
    "terraform-ls_${version}_windows_amd64.zip",
    "terraform-ls_${version}_windows_arm64.zip",
  ]
  rpm = [
    "terraform-ls-${version_linux}-1.aarch64.rpm",
    "terraform-ls-${version_linux}-1.armv7hl.rpm",
    "terraform-ls-${version_linux}-1.i386.rpm",
    "terraform-ls-${version_linux}-1.x86_64.rpm",
  ]
  deb = [
    "terraform-ls_${version_linux}-1_amd64.deb",
    "terraform-ls_${version_linux}-1_arm64.deb",
    "terraform-ls_${version_linux}-1_armhf.deb",
    "terraform-ls_${version_linux}-1_i386.deb",
  ]
}
