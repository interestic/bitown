import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "bitown",
  description: "Embeddable isometric city that grows with visitor clicks",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  );
}
