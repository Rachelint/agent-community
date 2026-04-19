import { marked } from 'marked';

// Dead-simple markdown renderer for issue bodies and comments. We rely
// on marked's built-in HTML escaping plus a conservative `breaks` option
// for GitHub-like newline handling. No custom sanitizer yet — all
// content is authored locally by the user or an agent running on the
// user's machine.
marked.setOptions({ breaks: true });

export function Markdown({ source }: { source: string }) {
  if (!source.trim()) {
    return <p className="text-xs italic text-muted">no description</p>;
  }
  const html = marked.parse(source) as string;
  return (
    <div
      className="markdown-body prose prose-sm max-w-none text-xs leading-relaxed"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
