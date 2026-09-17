import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Browser } from "@wailsio/runtime";

// Markdown renders package READMEs; links open in the system browser instead of
// navigating the app webview away.
export default function Markdown({ children }: { children: string }) {
  return (
    <div className="markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => (
            <a
              href={href}
              onClick={(e) => {
                e.preventDefault();
                if (href?.startsWith("http")) Browser.OpenURL(href);
              }}
            >
              {children}
            </a>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
