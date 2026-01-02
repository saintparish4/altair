'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { motion } from 'framer-motion'
import { Github, ExternalLink, Menu, X } from 'lucide-react'
import { useState } from 'react'

const navLinks = [
  { href: '/', label: 'Home' },
  { href: '/demo', label: 'Live Demo' },
  { href: '/docs', label: 'Documentation' },
]

export function Navbar() {
  const pathname = usePathname()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  return (
    <header className="fixed top-0 left-0 right-0 z-50">
      {/* Backdrop blur */}
      <div className="absolute inset-0 bg-altair-bg/80 backdrop-blur-md border-b border-altair-border" />

      <nav className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-3 group">
            <div className="relative w-8 h-8">
              {/* Logo mark - stylized "A" */}
              <svg viewBox="0 0 32 32" className="w-full h-full">
                <path
                  d="M16 4L4 28h6l2-4h8l2 4h6L16 4zm0 10l3 6h-6l3-6z"
                  fill="url(#logo-gradient)"
                />
                <defs>
                  <linearGradient id="logo-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stopColor="#22d3ee" />
                    <stop offset="100%" stopColor="#4ade80" />
                  </linearGradient>
                </defs>
              </svg>
              {/* Glow effect */}
              <div className="absolute inset-0 blur-lg bg-signal-cyan/30 opacity-0 group-hover:opacity-100 transition-opacity" />
            </div>
            <span className="font-display font-bold text-xl text-altair-text">
              Altair
            </span>
          </Link>

          {/* Desktop nav */}
          <div className="hidden md:flex items-center gap-1">
            {navLinks.map(({ href, label }) => {
              const isActive = pathname === href
              return (
                <Link
                  key={href}
                  href={href}
                  className={`
                    relative px-4 py-2 text-sm font-medium rounded-lg transition-colors
                    ${isActive 
                      ? 'text-signal-cyan' 
                      : 'text-altair-text-dim hover:text-altair-text'
                    }
                  `}
                >
                  {label}
                  {isActive && (
                    <motion.div
                      layoutId="nav-indicator"
                      className="absolute inset-0 bg-signal-cyan/10 rounded-lg border border-signal-cyan/20"
                      transition={{ type: 'spring', bounce: 0.2, duration: 0.6 }}
                    />
                  )}
                </Link>
              )
            })}
          </div>

          {/* Right side */}
          <div className="flex items-center gap-3">
            <a
              href="https://github.com/saintparish4/altair"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden sm:flex items-center gap-2 px-4 py-2 text-sm font-medium text-altair-text-dim 
                hover:text-altair-text bg-altair-surface border border-altair-border rounded-lg 
                hover:border-altair-muted transition-all"
            >
              <Github className="w-4 h-4" />
              <span>GitHub</span>
              <ExternalLink className="w-3 h-3 opacity-50" />
            </a>

            {/* Mobile menu button */}
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="md:hidden p-2 text-altair-text-dim hover:text-altair-text"
            >
              {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </button>
          </div>
        </div>

        {/* Mobile menu */}
        {mobileMenuOpen && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="md:hidden absolute top-full left-0 right-0 p-4 bg-altair-surface border-b border-altair-border"
          >
            <div className="space-y-1">
              {navLinks.map(({ href, label }) => {
                const isActive = pathname === href
                return (
                  <Link
                    key={href}
                    href={href}
                    onClick={() => setMobileMenuOpen(false)}
                    className={`
                      block px-4 py-3 text-sm font-medium rounded-lg transition-colors
                      ${isActive 
                        ? 'text-signal-cyan bg-signal-cyan/10' 
                        : 'text-altair-text-dim hover:text-altair-text hover:bg-altair-border/50'
                      }
                    `}
                  >
                    {label}
                  </Link>
                )
              })}
              <a
                href="https://github.com/saintparish4/altair"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 px-4 py-3 text-sm font-medium text-altair-text-dim 
                  hover:text-altair-text rounded-lg hover:bg-altair-border/50 transition-colors"
              >
                <Github className="w-4 h-4" />
                <span>View on GitHub</span>
              </a>
            </div>
          </motion.div>
        )}
      </nav>
    </header>
  )
}