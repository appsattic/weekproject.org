// IIFE
(function () {

  // the slider to listen to
  var slider = document.getElementById('slider')
  console.log(slider)

  // the place to display the percentage
  var display = document.getElementById('percentage')
  display.textContent = slider.value + '%'

  // when the slider changes, set the value
  slider.onchange = function(ev) {
    display.textContent = ev.target.value + '%'
  }

})()
