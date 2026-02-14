export interface SearchResult {
  id: number | string;
  type: "contact" | "note";
  title: string;
  subtitle?: string;
  contact_id?: string;
}

export interface SearchResponse {
  contacts: SearchResult[];
  notes: SearchResult[];
}
