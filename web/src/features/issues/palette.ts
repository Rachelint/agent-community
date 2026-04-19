// GitHub-style fixed palette. Users pick from these when creating a
// label. Full hex is stored; display uses the hex directly.
export const LABEL_PALETTE: { name: string; hex: string }[] = [
  { name: 'red',    hex: 'd73a4a' },
  { name: 'orange', hex: 'fb8500' },
  { name: 'yellow', hex: 'd4a017' },
  { name: 'green',  hex: '0e8a16' },
  { name: 'teal',   hex: '0e7490' },
  { name: 'blue',   hex: '0969da' },
  { name: 'purple', hex: '8250df' },
  { name: 'gray',   hex: '6e7781' },
];

// Rough luminance-based contrast pick for the label text.
export function labelTextColor(hex: string): string {
  const v = parseInt(hex, 16);
  const r = (v >> 16) & 0xff;
  const g = (v >> 8) & 0xff;
  const b = v & 0xff;
  // perceived brightness formula from W3C
  const brightness = (r * 299 + g * 587 + b * 114) / 1000;
  return brightness > 140 ? '#1f2328' : '#ffffff';
}
