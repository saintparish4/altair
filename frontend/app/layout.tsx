import type { Metadata } from "next";
import "./globals.css";


export const metadata: Metadata = {
  title: "Altair - P2P NAT Traversal Toolkit",
  description: "P2P NAT Traversal Toolkit",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-altair-bg">
        {/* Background effects */}
        <div className="fixed inset-0 grid-bg pointer-events-none" />
        <div className="fixed inset-0 bg-radial-glow pointer-events-none opacity-50" />

        {/* Main content */}
        <div className="relative z-10">{children}</div>
      </body>
    </html>
  );
}
