package cmd

import (
	"fmt"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ocmQuotaCostMetricDescription = "Openshift Cluster Manager Quota Costs"
	ocmQuotaCostMetricName        = "ocm_quota_cost"
	labelOrganizationID           = "organization_id"
	labelQuotaID                  = "quota_id"
	labelType                     = "type"
	valueConsumed                 = "consumed"
	valueAllowed                  = "allowed"
)

type ocmCollector struct {
	quotaCostClient *v1.QuotaCostClient
	quotas          *prometheus.Desc
}

func newOcmCollector(quotaCostClient *v1.QuotaCostClient) *ocmCollector {
	return &ocmCollector{
		quotaCostClient: quotaCostClient,
		quotas: prometheus.NewDesc(ocmQuotaCostMetricName,
			ocmQuotaCostMetricDescription,
			[]string{labelOrganizationID, labelQuotaID, labelType}, nil,
		),
	}
}

func (c *ocmCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.quotas
}

func (c *ocmCollector) Collect(ch chan<- prometheus.Metric) {

	quotasListResponse, err := c.quotaCostClient.List().
		Parameter("fetchRelatedResources", true).
		Send()

	if err != nil {
		fmt.Printf("[quota cost] failed to retrieve quota costs: %v\n", err)
		return
	}

	quotasListResponse.Items().Each(func(quota *v1.QuotaCost) bool {
		consumed := quota.Consumed()
		allowed := quota.Allowed()
		quotaID := quota.QuotaID()
		orgID := quota.OrganizationID()
		consumedMetric, err := prometheus.NewConstMetric(c.quotas, prometheus.GaugeValue, float64(consumed), orgID, quotaID, valueConsumed)
		if err == nil {
			ch <- consumedMetric
		} else {
			fmt.Printf("[quota cost] failed to create consumed metric: %v\n", err)
		}
		allowedMetric, err := prometheus.NewConstMetric(c.quotas, prometheus.GaugeValue, float64(allowed), orgID, quotaID, valueAllowed)
		if err == nil {
			ch <- allowedMetric
		} else {
			fmt.Printf("[quota cost] failed to create allowed metric: %v\n", err)
		}
		return true
	})
}
