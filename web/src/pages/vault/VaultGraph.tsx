import { useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, Typography, Button, Select, Space, Tag, theme } from "antd";
import { ArrowLeftOutlined, ShareAltOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { httpClient } from "@/api";
import NetworkGraph from "@/components/NetworkGraph";
import {
  vaultGraphQueryKey,
  vaultGraphURL,
} from "@/components/networkGraphQueryKey";
import type { VaultGraphFilters } from "@/components/networkGraphQueryKey";

const { Title, Text } = Typography;

// A force layout stops being readable long before it stops being renderable,
// so the default is a legibility choice. On a large vault it can leave most of
// the related contacts out, which is the reader's call to overrule, not ours.
const NODE_LIMITS = [400, 1000, 2500, 5000];
const DEFAULT_NODE_LIMIT = 1000;

interface VaultGraphFacetValue {
  value: string;
  label: string;
  count: number;
}

interface VaultGraphFacet {
  key: string;
  values: VaultGraphFacetValue[];
}

interface VaultGraphSummary {
  nodes?: { id: string }[];
  components?: number;
  isolated_contacts?: number;
  external_relationships?: number;
  truncated?: boolean;
  facets?: VaultGraphFacet[];
  filtered_out?: number;
}

export default function VaultGraph() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [limit, setLimit] = useState(DEFAULT_NODE_LIMIT);
  const [filters, setFilters] = useState<VaultGraphFilters>({});

  // Same query key as the canvas below, so the page and the drawing share one
  // request rather than each fetching the vault graph separately.
  const { data: summary } = useQuery({
    queryKey: vaultGraphQueryKey(vaultId, limit, filters),
    queryFn: async () => {
      const res = await httpClient.instance.get<{
        success: boolean;
        data: VaultGraphSummary;
      }>(vaultGraphURL(vaultId, limit, filters));
      return (res.data?.data ?? res.data) as VaultGraphSummary;
    },
  });

  const drawnContacts = summary?.nodes?.length ?? 0;
  const isolatedContacts = summary?.isolated_contacts ?? 0;
  const externalRelationships = summary?.external_relationships ?? 0;
  const filteredOut = summary?.filtered_out ?? 0;

  // The server omits facets nothing carries, so this renders a control only
  // where there is something to pick. On a vault where nobody has a gender set,
  // no gender control appears — and one appears by itself once they do.
  const facets = useMemo(() => summary?.facets ?? [], [summary?.facets]);
  const hasSelection = Object.values(filters).some(
    (values) => values.length > 0,
  );

  const selectFacet = (key: string, values: string[]) =>
    setFilters((current) => ({ ...current, [key]: values }));

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto", paddingBottom: 48 }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          marginBottom: 8,
        }}
      >
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <ShareAltOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>
          {t("vault.graph.title")}
        </Title>
      </div>
      <Text type="secondary" style={{ display: "block", marginBottom: 20 }}>
        {t("vault.graph.description")}
      </Text>

      <Card styles={{ body: { paddingTop: 12 } }}>
        <div
          style={{
            display: "flex",
            flexWrap: "wrap",
            justifyContent: "space-between",
            alignItems: "center",
            gap: 8,
            marginBottom: 12,
          }}
        >
          <Space size={[8, 8]} wrap>
            {facets.map((facet) => (
              <Select
                key={facet.key}
                mode="multiple"
                allowClear
                size="small"
                maxTagCount="responsive"
                // Responsive tags must measure against a stable width: an
                // auto-sized Select can oscillate between a tag and "+1".
                style={{ width: 220, maxWidth: "100%" }}
                value={[...(filters[facet.key] ?? [])]}
                onChange={(values: string[]) => selectFacet(facet.key, values)}
                placeholder={t(`vault.graph.facet.${facet.key}`)}
                aria-label={t(`vault.graph.facet.${facet.key}`)}
                options={facet.values.map((value) => ({
                  value: value.value,
                  label: `${value.label} (${value.count})`,
                }))}
              />
            ))}
            {hasSelection && (
              <Button size="small" type="link" onClick={() => setFilters({})}>
                {t("vault.graph.clearFilters")}
              </Button>
            )}
          </Space>

          <Space size={8}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t("vault.graph.limitLabel")}
            </Text>
            <Select
              size="small"
              value={limit}
              onChange={setLimit}
              style={{ width: 110 }}
              aria-label={t("vault.graph.limitLabel")}
              options={NODE_LIMITS.map((value) => ({
                value,
                label: t("vault.graph.limitOption", { count: value }),
              }))}
            />
          </Space>
        </div>

        <NetworkGraph
          vaultId={vaultId}
          limit={limit}
          filters={filters}
          height={620}
          emptyDescription={t(
            hasSelection ? "vault.graph.emptyFiltered" : "vault.graph.empty",
          )}
        />

        {/* What is not on the canvas matters as much as what is, so the counts
            that were left out are stated rather than silently dropped. */}
        <Space size={[8, 8]} wrap style={{ marginTop: 4 }}>
          <Tag color="blue">
            {t("vault.graph.drawn", { count: drawnContacts })}
          </Tag>
          {(summary?.components ?? 0) > 1 && (
            <Tag>
              {t("vault.graph.clusters", { count: summary?.components ?? 0 })}
            </Tag>
          )}
          {filteredOut > 0 && (
            <Tag color="purple">
              {t("vault.graph.filteredOut", { count: filteredOut })}
            </Tag>
          )}
          {isolatedContacts > 0 && (
            <Tag>{t("vault.graph.isolated", { count: isolatedContacts })}</Tag>
          )}
          {externalRelationships > 0 && (
            <Tag>
              {t("vault.graph.external", { count: externalRelationships })}
            </Tag>
          )}
          {summary?.truncated && (
            <Tag color="orange">{t("vault.graph.truncated")}</Tag>
          )}
        </Space>
      </Card>
    </div>
  );
}
