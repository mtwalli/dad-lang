// Small touches for the docs theme (dad.css). Everything here is optional:
// without it the page reads the same, just without the light and the reveal.
(function () {
  "use strict"
  var still = window.matchMedia("(prefers-reduced-motion: reduce)").matches

  // The hero lattice lights up like gold leaf around the pointer.
  var hero = document.querySelector(".dad-hero")
  if (hero && !still && window.matchMedia("(hover: hover)").matches) {
    var frame = 0
    hero.addEventListener("pointermove", function (e) {
      if (frame) return
      frame = requestAnimationFrame(function () {
        frame = 0
        var r = hero.getBoundingClientRect()
        hero.style.setProperty("--mx", ((e.clientX - r.left) / r.width * 100).toFixed(1) + "%")
        hero.style.setProperty("--my", ((e.clientY - r.top) / r.height * 100).toFixed(1) + "%")
      })
    })
  }

  // The roadmap's completed phases join up once the timeline comes into view.
  var timelines = document.querySelectorAll(".dad-timeline")
  timelines.forEach(function (ol) {
    ol.querySelectorAll("li.is-done").forEach(function (li, n) {
      li.style.setProperty("--n", n)
    })
  })
  if (!("IntersectionObserver" in window) || still) {
    timelines.forEach(function (ol) { ol.classList.add("is-seen") })
    return
  }
  var seen = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (!entry.isIntersecting) return
      entry.target.classList.add("is-seen")
      seen.unobserve(entry.target)
    })
  }, { threshold: 0.4 })
  timelines.forEach(function (ol) { seen.observe(ol) })
})()
