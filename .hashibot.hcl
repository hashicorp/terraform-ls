behavior "remove_labels_on_reply" "remove_stale" {
  labels               = ["waiting-response", "stale"]
  only_non_maintainers = true
}

poll "closed_issue_locker" "locker" {
  schedule                      = "0 3 * * *" # daily
  closed_for                    = "720h" # 30 days
  no_comment_if_no_activity_for = "1440h" # 60 days
  max_issues                    = 500
  sleep_between_issues          = "5s"

  message = <<-EOF
    I'm going to lock this issue because it has been closed for _30 days_ ⏳. This helps our maintainers find and focus on the active issues.
    If you have found a problem that seems similar to this, please open a new issue and complete the issue template so we can capture all the context necessary to investigate further.
  EOF
}

poll "stale_issue_closer" "closer" {
    schedule = "0 3 * * *" # daily
    labels = ["stale", "waiting-response"]
    no_reply_in_last = "2160h" # 90 days
    max_issues = 500
    sleep_between_issues = "5s"
    message = <<-EOF
    I'm going to close this issue due to inactivity (_90 days_ without response ⏳ ). This helps our maintainers find and focus on the active issues.
    If you have found a problem that seems similar to this, please open a new issue and complete the issue template so we can capture all the context necessary to investigate further.
    EOF
}
