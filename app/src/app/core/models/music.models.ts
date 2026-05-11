export interface MusicEntry {
  id: string;
  title: string;
  artist: string;
  album?: string;
  spotify_url?: string;
  youtube_url?: string;
  mostly_listened_on: string;
  notes?: string;
  sort_order: number;
  status: 'draft' | 'published' | 'archived';
}
