data "netapp-ontap_protocols_nfs_export_policy_rule_data_source" "rule" {
  cx_profile_name = "cluster4"
  vserver = "automation"
  export_policy_name = "test"
  index = 2
}