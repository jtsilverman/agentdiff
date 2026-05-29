/* agentdiff — shared site chrome behaviors: nav scroll state, reveal-on-scroll,
   FAQ accordion, footer year. Tiny, dependency-free. */
(function () {
  'use strict';
  function ready(fn) {
    if (document.readyState !== 'loading') fn();
    else document.addEventListener('DOMContentLoaded', fn);
  }
  ready(function () {
    // nav scroll state
    const nav = document.querySelector('.nav');
    if (nav) {
      const onScroll = function () { nav.classList.toggle('scrolled', window.scrollY > 12); };
      onScroll();
      window.addEventListener('scroll', onScroll, { passive: true });
    }

    // reveal on scroll
    const reveals = document.querySelectorAll('.reveal');
    if ('IntersectionObserver' in window && reveals.length) {
      const io = new IntersectionObserver(function (entries) {
        entries.forEach(function (e) {
          if (e.isIntersecting) { e.target.classList.add('in'); io.unobserve(e.target); }
        });
      }, { threshold: 0.14, rootMargin: '0px 0px -8% 0px' });
      reveals.forEach(function (r) { io.observe(r); });
    } else {
      reveals.forEach(function (r) { r.classList.add('in'); });
    }

    // FAQ accordion
    document.querySelectorAll('.faq-item').forEach(function (item) {
      const q = item.querySelector('.faq-q');
      if (!q) return;
      q.addEventListener('click', function () {
        const open = item.classList.contains('open');
        item.classList.toggle('open', !open);
        q.setAttribute('aria-expanded', String(!open));
      });
    });

    // footer year
    document.querySelectorAll('[data-year]').forEach(function (e) { e.textContent = new Date().getFullYear(); });
  });
})();
