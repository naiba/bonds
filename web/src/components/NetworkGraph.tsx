import { useRef, useEffect, useState, useCallback } from "react";
import * as d3 from "d3";
import { Spin, Empty, Typography, theme } from "antd";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { httpClient } from "@/api";

const { Text } = Typography;

interface GraphNode extends d3.SimulationNodeDatum {
  id: string;
  label: string;
  is_center: boolean;
}

interface GraphEdge {
  source: string | GraphNode;
  target: string | GraphNode;
  type: string;
}

interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

interface KinshipResult {
  degree: number;
  path: string[];
}

interface NetworkGraphProps {
  vaultId: string;
  contactId: string;
}

export default function NetworkGraph({ vaultId, contactId }: NetworkGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
  const [kinship, setKinship] = useState<KinshipResult | null>(null);
  const [kinshipLoading, setKinshipLoading] = useState(false);
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const { data: graphData, isLoading: loading, isError: error } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "graph"],
    queryFn: async () => {
      const res = await httpClient.instance.get<{ success: boolean; data: GraphData }>(
        `/vaults/${vaultId}/contacts/${contactId}/relationships/graph`
      );
      const data = res.data?.data ?? res.data;
      if (data && "nodes" in data && "edges" in data) {
        return data;
      }
      return { nodes: [], edges: [] } as GraphData;
    },
  });

  const fetchKinship = useCallback(
    (nodeIds: string[]) => {
      if (nodeIds.length !== 2) {
        setKinship(null);
        return;
      }

      setKinshipLoading(true);

      httpClient.instance
        .get<{ success: boolean; data: KinshipResult }>(
          `/vaults/${vaultId}/contacts/${nodeIds[0]}/relationships/kinship/${nodeIds[1]}`
        )
        .then((res) => {
          const data = res.data?.data ?? res.data;
          if (data && typeof data.degree === "number") {
            setKinship(data);
          } else {
            setKinship(null);
          }
        })
        .catch(() => {
          setKinship(null);
        })
        .finally(() => {
          setKinshipLoading(false);
        });
    },
    [vaultId]
  );

  const getNodeId = useCallback((node: string | GraphNode): string => {
    return typeof node === "string" ? node : node.id;
  }, []);

  useEffect(() => {
    if (!graphData || !svgRef.current || !containerRef.current) return;
    if (graphData.nodes.length === 0) return;

    const container = containerRef.current;
    const width = container.clientWidth;
    const height = 500;
    const isMobile = width < 600;

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();
    svg.attr("viewBox", `0 0 ${width} ${height}`);

    const isDark = document.documentElement.classList.contains("dark");
    const textColor = isDark ? "#e5e7eb" : "#374151";
    const edgeColor = isDark ? "#4b5563" : "#d1d5db";
    const centerColor = token.colorPrimary;
    const nodeColor = isDark ? "#6b7280" : "#9ca3af";
    const hoverEdgeColor = isDark ? "#93c5fd" : "#3b82f6";

    const nodeRadius = isMobile ? 6 : 8;
    const centerNodeRadius = isMobile ? 9 : 12;
    const chargeStrength = isMobile ? -200 : -400;
    const linkDistance = isMobile ? 80 : 120;

    const nodes: GraphNode[] = graphData.nodes.map((n) => ({ ...n }));
    const edges: GraphEdge[] = graphData.edges.map((e) => ({ ...e }));

    const g = svg.append("g");

    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.5, 3])
      .on("zoom", (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        g.attr("transform", event.transform.toString());
      });
    svg.call(zoom);

    svg.on("click", (event: MouseEvent) => {
      if (event.target === svgRef.current) {
        setSelectedNodes([]);
        fetchKinship([]);
      }
    });

    g.append("defs")
      .append("marker")
      .attr("id", "arrowhead")
      .attr("viewBox", "0 -5 10 10")
      .attr("refX", 20)
      .attr("refY", 0)
      .attr("markerWidth", 6)
      .attr("markerHeight", 6)
      .attr("orient", "auto")
      .append("path")
      .attr("d", "M0,-5L10,0L0,5")
      .attr("fill", edgeColor);

    const link = g
      .append("g")
      .selectAll<SVGLineElement, GraphEdge>("line")
      .data(edges)
      .join("line")
      .attr("stroke", edgeColor)
      .attr("stroke-opacity", 0.3)
      .attr("stroke-width", 1.5);

    const edgeLabel = g
      .append("g")
      .selectAll<SVGTextElement, GraphEdge>("text")
      .data(edges)
      .join("text")
      .text((d) => d.type)
      .attr("font-size", 10)
      .attr("fill", textColor)
      .attr("text-anchor", "middle")
      .attr("opacity", 0)
      .attr("pointer-events", "none");

    const node = g
      .append("g")
      .selectAll<SVGGElement, GraphNode>("g")
      .data(nodes)
      .join("g")
      .style("cursor", "pointer");

    node
      .append("circle")
      .attr("r", (d) => (d.is_center ? centerNodeRadius : nodeRadius))
      .attr("fill", (d) => (d.is_center ? centerColor : nodeColor))
      .attr("stroke", (d) => (d.is_center ? centerColor : "transparent"))
      .attr("stroke-width", 2)
      .attr("stroke-opacity", 0.3);

    node
      .append("text")
      .text((d) => d.label)
      .attr("dy", (d) => (d.is_center ? centerNodeRadius + 14 : nodeRadius + 14))
      .attr("text-anchor", "middle")
      .attr("font-size", isMobile ? 10 : 12)
      .attr("fill", textColor)
      .attr("pointer-events", "none");

    node
      .on("mouseenter", (_event, d) => {
        link
          .attr("stroke", (l) => {
            const srcId = getNodeId(l.source);
            const tgtId = getNodeId(l.target);
            return srcId === d.id || tgtId === d.id ? hoverEdgeColor : edgeColor;
          })
          .attr("stroke-opacity", (l) => {
            const srcId = getNodeId(l.source);
            const tgtId = getNodeId(l.target);
            return srcId === d.id || tgtId === d.id ? 0.8 : 0.3;
          })
          .attr("stroke-width", (l) => {
            const srcId = getNodeId(l.source);
            const tgtId = getNodeId(l.target);
            return srcId === d.id || tgtId === d.id ? 2.5 : 1.5;
          });

        edgeLabel.attr("opacity", (l) => {
          const srcId = getNodeId(l.source);
          const tgtId = getNodeId(l.target);
          return srcId === d.id || tgtId === d.id ? 1 : 0;
        });
      })
      .on("mouseleave", () => {
        link
          .attr("stroke", edgeColor)
          .attr("stroke-opacity", 0.3)
          .attr("stroke-width", 1.5);
        edgeLabel.attr("opacity", 0);
      });

    node.on("click", (event, d) => {
      event.stopPropagation();

      if (event.metaKey || event.ctrlKey) {
        navigate(`/vaults/${vaultId}/contacts/${d.id}`);
        return;
      }

      setSelectedNodes((prev) => {
        let next: string[];
        if (prev.includes(d.id)) {
          next = prev.filter((id) => id !== d.id);
        } else if (prev.length >= 2) {
          next = [d.id];
        } else {
          next = [...prev, d.id];
        }
        fetchKinship(next);
        return next;
      });
    });

    node.on("dblclick", (event, d) => {
      event.stopPropagation();
      event.preventDefault();
      navigate(`/vaults/${vaultId}/contacts/${d.id}`);
    });

    const drag = d3
      .drag<SVGGElement, GraphNode>()
      .on("start", (event: d3.D3DragEvent<SVGGElement, GraphNode, GraphNode>, d) => {
        if (!event.active) simulation.alphaTarget(0.3).restart();
        d.fx = d.x;
        d.fy = d.y;
      })
      .on("drag", (event: d3.D3DragEvent<SVGGElement, GraphNode, GraphNode>, d) => {
        d.fx = event.x;
        d.fy = event.y;
      })
      .on("end", (event: d3.D3DragEvent<SVGGElement, GraphNode, GraphNode>, d) => {
        if (!event.active) simulation.alphaTarget(0);
        d.fx = null;
        d.fy = null;
      });
    node.call(drag);

    const simulation = d3
      .forceSimulation(nodes)
      .force(
        "link",
        d3
          .forceLink<GraphNode, GraphEdge>(edges)
          .id((d) => d.id)
          .distance(linkDistance)
      )
      .force("charge", d3.forceManyBody().strength(chargeStrength))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("collision", d3.forceCollide().radius(nodeRadius + 10));

    simulation.on("tick", () => {
      link
        .attr("x1", (d) => (d.source as GraphNode).x ?? 0)
        .attr("y1", (d) => (d.source as GraphNode).y ?? 0)
        .attr("x2", (d) => (d.target as GraphNode).x ?? 0)
        .attr("y2", (d) => (d.target as GraphNode).y ?? 0);

      edgeLabel
        .attr("x", (d) => (((d.source as GraphNode).x ?? 0) + ((d.target as GraphNode).x ?? 0)) / 2)
        .attr("y", (d) => (((d.source as GraphNode).y ?? 0) + ((d.target as GraphNode).y ?? 0)) / 2 - 6);

      node.attr("transform", (d) => `translate(${d.x ?? 0},${d.y ?? 0})`);
    });

    return () => {
      simulation.stop();
    };
  }, [graphData, token.colorPrimary, navigate, vaultId, getNodeId, fetchKinship]);

  useEffect(() => {
    if (!svgRef.current || !graphData) return;
    const svg = d3.select(svgRef.current);

    const isDark = document.documentElement.classList.contains("dark");
    const edgeColor = isDark ? "#4b5563" : "#d1d5db";
    const centerColor = token.colorPrimary;
    const nodeColor = isDark ? "#6b7280" : "#9ca3af";
    const selectedColor = "#f59e0b";
    const highlightEdgeColor = "#ef4444";

    svg.selectAll<SVGCircleElement, GraphNode>("circle").attr("fill", (d) => {
      if (selectedNodes.includes(d.id)) return selectedColor;
      if (d.is_center) return centerColor;
      return nodeColor;
    });

    if (kinship && kinship.path && kinship.path.length > 1) {
      const pathSet = new Set(kinship.path);
      svg
        .selectAll<SVGLineElement, GraphEdge>("line")
        .attr("stroke", (l) => {
          const srcId = getNodeId(l.source);
          const tgtId = getNodeId(l.target);
          if (pathSet.has(srcId) && pathSet.has(tgtId)) {
            for (let i = 0; i < kinship.path.length - 1; i++) {
              if (
                (kinship.path[i] === srcId && kinship.path[i + 1] === tgtId) ||
                (kinship.path[i] === tgtId && kinship.path[i + 1] === srcId)
              ) {
                return highlightEdgeColor;
              }
            }
          }
          return edgeColor;
        })
        .attr("stroke-opacity", (l) => {
          const srcId = getNodeId(l.source);
          const tgtId = getNodeId(l.target);
          if (pathSet.has(srcId) && pathSet.has(tgtId)) return 0.9;
          return 0.3;
        })
        .attr("stroke-width", (l) => {
          const srcId = getNodeId(l.source);
          const tgtId = getNodeId(l.target);
          if (pathSet.has(srcId) && pathSet.has(tgtId)) return 3;
          return 1.5;
        });
    } else {
      svg
        .selectAll<SVGLineElement, GraphEdge>("line")
        .attr("stroke", edgeColor)
        .attr("stroke-opacity", 0.3)
        .attr("stroke-width", 1.5);
    }
  }, [selectedNodes, kinship, graphData, token.colorPrimary, getNodeId]);

  if (loading) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
        <Spin />
      </div>
    );
  }

  if (error || !graphData || graphData.nodes.length === 0) {
    return <Empty description={t("modules.relationships.graph_empty")} />;
  }

  return (
    <div ref={containerRef} style={{ position: "relative" }}>
      <svg
        ref={svgRef}
        style={{
          width: "100%",
          height: 500,
          background: token.colorBgContainer,
          borderRadius: token.borderRadiusLG,
          border: `1px solid ${token.colorBorderSecondary}`,
        }}
      />

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          padding: "8px 12px",
          fontSize: 12,
          color: token.colorTextSecondary,
        }}
      >
        <Text type="secondary" style={{ fontSize: 12 }}>
          {selectedNodes.length === 0 && t("modules.relationships.click_to_calculate")}
          {selectedNodes.length === 1 && t("modules.relationships.click_to_calculate")}
          {selectedNodes.length === 2 && kinshipLoading && <Spin size="small" />}
          {selectedNodes.length === 2 &&
            !kinshipLoading &&
            kinship &&
            t("modules.relationships.kinship_degree", { degree: kinship.degree })}
          {selectedNodes.length === 2 &&
            !kinshipLoading &&
            !kinship &&
            t("modules.relationships.kinship_no_path")}
        </Text>
        <Text type="secondary" style={{ fontSize: 11 }}>
          Ctrl+Click / Double-click to navigate
        </Text>
      </div>
    </div>
  );
}
