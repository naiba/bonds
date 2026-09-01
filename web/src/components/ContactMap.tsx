import { useEffect, useMemo, useRef, useState } from "react";
import * as d3 from "d3";
import { Empty, Skeleton, Typography, theme } from "antd";
import { useTranslation } from "react-i18next";
import type { AddressAttribution, MapCountryItem, MapPoint } from "@/api";
import { useElementWidth } from "@/hooks/useElementWidth";
import { fitProjectionToPoints } from "@/components/mapProjection";
import { matchCountryName } from "@/utils/countryNames";

const { Text } = Typography;

type WorldFeature = {
  type: "Feature";
  id?: string | number;
  properties: { name: string };
  geometry: { type: "MultiPolygon"; coordinates: number[][][][] };
};

type WorldData = { type: "FeatureCollection"; features: WorldFeature[] };

type HoverState = {
  x: number;
  y: number;
  title: string;
  lines: string[];
};

/** One dot: an exact location, and everyone recorded as living there. */
type MapDot = {
  key: string;
  latitude: number;
  longitude: number;
  city: string;
  country: string;
  contacts: { contact_id?: string; contact_name?: string }[];
  addresses: number;
};

type Props = {
  points: MapPoint[];
  countries: MapCountryItem[];
  attribution?: AddressAttribution[];
  height?: number;
  onSelectContact?: (contactId: string) => void;
  /** True while the report is being fetched, so absent data is not mistaken for an empty vault. */
  loading?: boolean;
};

// The world outline is ~100KB gzipped and only the reports page needs it, so it
// is fetched on demand rather than bundled into the initial download.
let worldPromise: Promise<WorldData> | null = null;
function loadWorld(): Promise<WorldData> {
  if (!worldPromise) {
    worldPromise = import("@/assets/world-countries.geo.json").then(
      (module) => (module.default ?? module) as unknown as WorldData,
    );
  }
  return worldPromise;
}

export default function ContactMap({ points, countries, attribution = [], height = 420, onSelectContact , loading = false }: Props) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const svgRef = useRef<SVGSVGElement>(null);
  const [world, setWorld] = useState<WorldData | null>(null);
  const [hover, setHover] = useState<HoverState | null>(null);
  const { ref: containerRef, nodeRef: containerNodeRef, width } = useElementWidth();

  useEffect(() => {
    let active = true;
    loadWorld().then((data) => {
      if (active) setWorld(data);
    });
    return () => {
      active = false;
    };
  }, []);

  // Countries are matched to map features by name, which is why this is keyed on
  // a normalised form rather than the raw string a user typed into an address.
  const countsByFeature = useMemo(() => {
    const counts = new Map<string, MapCountryItem & { contactIds: Set<string> }>();
    for (const country of countries) {
      const key = matchCountryName(country.country ?? "");
      if (!key) continue;
      const existing = counts.get(key);
      if (existing) {
        // Two spellings of one country ("USA" and "United States") land on the
        // same feature. Addresses add up, but people must be UNIONED — someone
        // with an address recorded under each spelling is still one person.
        for (const id of country.contact_ids ?? []) existing.contactIds.add(id);
        counts.set(key, {
          ...existing,
          address_count: (existing.address_count ?? 0) + (country.address_count ?? 0),
          contact_count: existing.contactIds.size > 0
            ? existing.contactIds.size
            : (existing.contact_count ?? 0) + (country.contact_count ?? 0),
          geocoded: (existing.geocoded ?? 0) + (country.geocoded ?? 0),
        });
      } else {
        counts.set(key, { ...country, contactIds: new Set(country.contact_ids ?? []) });
      }
    }
    return counts;
  }, [countries]);

  // Several addresses routinely resolve to one coordinate — a shared house, or a
  // geocoder that only knew the city. Drawing them as separate circles stacks
  // them into an unreadable blob, so they become one dot sized by its occupants.
  const dots = useMemo<MapDot[]>(() => {
    const merged = new Map<string, MapDot>();
    for (const point of points) {
      const latitude = point.latitude ?? 0;
      const longitude = point.longitude ?? 0;
      // ~11m of precision: close enough to be the same doorstep.
      const key = `${latitude.toFixed(4)},${longitude.toFixed(4)}`;
      const existing = merged.get(key);
      if (existing) {
        existing.addresses += 1;
        for (const contact of point.contacts ?? []) {
          if (!existing.contacts.some((c) => c.contact_id === contact.contact_id)) {
            existing.contacts.push(contact);
          }
        }
        if (!existing.city && point.city) existing.city = point.city;
      } else {
        merged.set(key, {
          key,
          latitude,
          longitude,
          city: point.city ?? "",
          country: point.country ?? "",
          contacts: [...(point.contacts ?? [])],
          addresses: 1,
        });
      }
    }
    return [...merged.values()];
  }, [points]);

  const maxCountry = useMemo(
    () => Math.max(1, ...[...countsByFeature.values()].map((c) => c.address_count ?? 0)),
    [countsByFeature],
  );

  useEffect(() => {
    if (!world || !svgRef.current || width <= 0) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();
    svg.attr("viewBox", `0 0 ${width} ${height}`);

    const root = svg.append("g");
    const projection = d3.geoMercator();
    const path = d3.geoPath(projection);

    // Fit to what the vault actually contains: a UK-only address book gets a map
    // of the UK, not a world map with one dot on it. Points win over countries
    // because they are the more precise answer to the same question.
    const highlighted = world.features.filter((feature) =>
      countsByFeature.has(matchCountryName(feature.properties.name)),
    );
    const padding = 12;
    const extent: [[number, number], [number, number]] = [
      [padding, padding],
      [width - padding, height - padding],
    ];
    if (dots.length > 0) {
      fitProjectionToPoints(projection, dots, width, height, padding);
    } else if (highlighted.length > 0) {
      projection.fitExtent(extent, {
        type: "FeatureCollection",
        features: highlighted,
      } as unknown as d3.GeoPermissibleObjects);
    } else {
      projection.fitExtent(extent, world as unknown as d3.GeoPermissibleObjects);
    }

    // The heat is carried by opacity over one colour rather than by a colour
    // ramp, so that every shaded country still reads as "has people in it"
    // against the flat grey of the ones that do not — and so the ramp never
    // gets dark enough to swallow the pins drawn on top of it. Square-rooted
    // because one dominant country would otherwise flatten everywhere else to
    // nothing.
    const heat = d3
      .scaleSqrt()
      .domain([0, Math.max(1, maxCountry)])
      .range([0.18, 0.62])
      .clamp(true);

    root
      .append("g")
      .selectAll("path")
      .data(world.features)
      .join("path")
      .attr("d", (feature) => path(feature as unknown as d3.GeoPermissibleObjects))
      .attr("fill", (feature) =>
        countsByFeature.has(matchCountryName(feature.properties.name))
          ? token.colorPrimary
          : token.colorFillSecondary,
      )
      .attr("fill-opacity", (feature) => {
        const match = countsByFeature.get(matchCountryName(feature.properties.name));
        return match ? heat(match.address_count ?? 0) : 1;
      })
      .attr("stroke", token.colorBorderSecondary)
      .attr("stroke-width", 0.5)
      .style("cursor", (feature) =>
        countsByFeature.has(matchCountryName(feature.properties.name)) ? "pointer" : "default",
      )
      .on("mousemove", (event: MouseEvent, feature) => {
        const match = countsByFeature.get(matchCountryName(feature.properties.name));
        if (!match) {
          setHover(null);
          return;
        }
        const [x, y] = d3.pointer(event, containerNodeRef.current);
        setHover({
          x,
          y,
          title: match.country || feature.properties.name,
          lines: [
            t("vault.reports.map.address_count", { count: match.address_count ?? 0 }),
            t("vault.reports.map.contact_count", { count: match.contact_count ?? 0 }),
          ],
        });
      })
      .on("mouseleave", () => setHover(null));

    // Bigger dots for shared addresses, on a square-root scale so area rather
    // than radius tracks the number of people.
    const radius = d3
      .scaleSqrt()
      .domain([1, Math.max(1, ...dots.map((dot) => dot.contacts.length || 1))])
      .range([4, 13])
      .clamp(true);

    root
      .append("g")
      .selectAll("circle")
      .data(dots, (dot) => (dot as MapDot).key)
      .join("circle")
      .attr("cx", (dot) => projection([dot.longitude, dot.latitude])?.[0] ?? 0)
      .attr("cy", (dot) => projection([dot.longitude, dot.latitude])?.[1] ?? 0)
      .attr("r", (dot) => radius(dot.contacts.length || 1))
      .attr("fill", token.colorInfo)
      .attr("fill-opacity", 0.95)
      .attr("stroke", token.colorBgContainer)
      .attr("stroke-width", 1.5)
      .style("cursor", "pointer")
      .on("mousemove", (event: MouseEvent, dot) => {
        const [x, y] = d3.pointer(event, containerNodeRef.current);
        const names = dot.contacts.map((c) => c.contact_name ?? "").filter(Boolean);
        setHover({
          x,
          y,
          title: [dot.city, dot.country].filter(Boolean).join(", "),
          // Long lists are trimmed: a tooltip is a glance, not a directory.
          lines: names.length > 6 ? [...names.slice(0, 6), `+${names.length - 6}`] : names,
        });
      })
      .on("mouseleave", () => setHover(null))
      .on("click", (_event: MouseEvent, dot) => {
        if (dot.contacts.length === 1 && dot.contacts[0].contact_id) {
          onSelectContact?.(dot.contacts[0].contact_id);
        }
      });

    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([1, 24])
      .on("zoom", (event) => {
        root.attr("transform", event.transform.toString());
        // Keep dots and borders the same visual size however far in you go.
        root.selectAll<SVGCircleElement, MapDot>("circle").attr("r", (dot) => radius(dot.contacts.length || 1) / event.transform.k);
        root.selectAll("path").attr("stroke-width", 0.5 / event.transform.k);
      });
    svg.call(zoom);

    return () => {
      svg.on(".zoom", null);
    };
  }, [
    world,
    width,
    height,
    dots,
    containerNodeRef,
    countsByFeature,
    maxCountry,
    onSelectContact,
    t,
    token.colorPrimary,
    token.colorInfo,
    token.colorFillSecondary,
    token.colorBorderSecondary,
    token.colorBgContainer,
  ]);

  if (countries.length === 0) {
    return loading ? (
      <Skeleton.Node active style={{ width: "100%", height }} />
    ) : (
      <Empty description={t("vault.reports.map.no_data")} image={Empty.PRESENTED_IMAGE_SIMPLE} />
    );
  }

  return (
    <div ref={containerRef} style={{ position: "relative", width: "100%" }}>
      {!world && <Skeleton.Node active style={{ width: "100%", height }} />}
      <svg
        ref={svgRef}
        role="img"
        aria-label={t("vault.reports.map.aria_label")}
        style={{
          width: "100%",
          height,
          display: world ? "block" : "none",
          borderRadius: token.borderRadiusLG,
          background: token.colorBgLayout,
          touchAction: "none",
        }}
      />
      {hover && (
        <div
          style={{
            position: "absolute",
            left: Math.min(hover.x + 12, Math.max(0, width - 200)),
            top: hover.y + 12,
            pointerEvents: "none",
            background: token.colorBgElevated,
            border: `1px solid ${token.colorBorderSecondary}`,
            borderRadius: token.borderRadius,
            boxShadow: token.boxShadowSecondary,
            padding: "6px 10px",
            maxWidth: 220,
            zIndex: 2,
          }}
        >
          <div style={{ fontWeight: 600, fontSize: 13 }}>{hover.title}</div>
          {hover.lines.map((line, index) => (
            <div key={index} style={{ fontSize: 12, color: token.colorTextSecondary }}>
              {line}
            </div>
          ))}
        </div>
      )}
      <Text type="secondary" style={{ fontSize: 12, display: "block", marginTop: 8 }}>
        {t("vault.reports.map.hint")} {attribution.map((credit, index) => (
          <span key={credit.url}>
            {index > 0 && " · "}
            <a href={credit.url} target="_blank" rel="noopener noreferrer">
              {credit.label}
            </a>
          </span>
        ))}
      </Text>
    </div>
  );
}
