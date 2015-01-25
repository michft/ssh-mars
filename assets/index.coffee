Util =
  id: (id) ->
    document.getElementById(id)

  element: (tagName, attrs) ->
    el = document.createElement(tagName)
    el.setAttribute(k, v) for k, v of attrs
    el.inject = (parent) -> parent.appendChild(el)
    el

  postForm: (path, data, success, error) ->
    params = []
    for k, v of data
      params.push(encodeURIComponent(k) + '=' + encodeURIComponent(v))
    
    req = new XMLHttpRequest()
    req.onload = success
    req.onerror = error
    
    req.open('post', path)
    req.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded')
    req.send(params.join('&'))

  csrfToken: () ->
    Util.id('csrf-token').getAttribute('value')

  myFingerprint: () ->
    Util.id('my-fingerprint').getAttribute('value')

Nav =
  init: () ->
    Util.id('signout-link').addEventListener 'click', Nav.signOut
    Util.id('delete-link').addEventListener 'click', Nav.deleteAccount

    window.addEventListener('popstate', Nav.updateVisiblePage)

    for el in document.getElementsByClassName('sp-link')
      el.addEventListener('click', Nav.navigateSubPage)
  
  updateVisiblePage: () ->
    for el in Util.id('explain').childNodes
      el.hidden = true

    page = location.pathname.match(/\/(.*)/)[1]
    page = 'intro' if page == ''
    Util.id(page).hidden = false

  navigateSubPage: (e) ->
    e.preventDefault()
    href = e.target.href || e.target.parentNode.href
    history.pushState(null, null, href)
    Nav.updateVisiblePage()

  signOut: () ->
    form = Util.element 'form',
        action: 'signout'
        method: 'post'
      .inject(document.body)

    Util.element 'input',
        type: 'hidden'
        name: 'csrf_token'
        value: Util.csrfToken()
      .inject(form)

    form.submit()

  deleteAccount: () ->
    form = Util.element 'form',
        action: 'delete-account'
        method: 'post'
      .inject(document.body)

    Util.element 'input',
        type: 'hidden'
        name: 'csrf_token'
        value: Util.csrfToken()
      .inject(form)

    form.submit()

Pin =
  init: (globe) ->
    pin =
      well: Util.id('pinwell')
      drag: Util.id('pindrag')
      globe: globe
      offset: new THREE.Vector2(0, 0)

    Pin.addEvents(pin)

    pin

  eventModes:
    rest: [
      ['well', 'mouseenter', 'wellEnter']
      ['well', 'mouseleave', 'wellLeave']
      ['well', 'mousedown', 'wellDown']]
    wellPressed: [
      ['well', 'mousemove', 'wellPull']
      ['well', 'mouseleave', 'dragStart']
      ['well', 'mouseup', 'wellUp']]
    dragVoid: [
      ['document', 'mousemove', 'dragMove']
      ['document', 'mouseup', 'dragReset']]
    dragGlobe: [
      ['document', 'mousemove', 'globeMove']
      ['document', 'mouseup', 'globeUp']]

  addEvents: (pin) ->
    pin.events =
      wellEnter: () -> pin.well.classList.add('hover')
      wellLeave: () -> pin.well.classList.remove('hover')

      wellDown: (e) ->
        pin.well.style.cursor = 'grabbing'
        pin.offset = Pin.wellOffset(pin.well, e)
        Pin.transitionMode(pin, 'wellPressed')

      wellUp: () ->
        pin.well.style.cursor = null
        Pin.transitionMode(pin, 'rest')

      wellPull: (e) ->
        dist = pin.offset.distanceTo(Pin.wellOffset(pin.well, e))
        pin.events.dragStart(e) if dist > 10

      dragStart: (e) ->
        pin.events.wellLeave()
        pin.events.wellUp()
        pin.well.classList.add('empty')
        pin.events.dragMove(e)
        pin.drag.hidden = false
        pin.drag.style.transformOrigin = pin.offset.x + 'px ' + pin.offset.y + 'px'

        fingerprint = Util.myFingerprint()
        gl = pin.globe.gl
        glpin = gl.pins.fingerprints[fingerprint]
        if glpin?
          gl.pins.fingerprints[fingerprint] = null
          gl.pins.remove(glpin)

        Pin.transitionMode(pin, 'dragVoid')

      dragMove: (e) ->
        pin.drag.style.left = (e.clientX - pin.offset.x) + 'px'
        pin.drag.style.top  = (e.clientY - pin.offset.y) + 'px'

        globeOffset = Pin.globeOffset(pin.globe.container, e)
        dist = Globe.glMouse(globeOffset).length()

        easing = (x) -> x * x
        scale = Pin.interpolate([1.54, 0.87], [1, 0.1], easing, dist)
        Pin.scalePin(pin.drag, scale)

        pos = Globe.raycast(pin.globe.gl, Pin.nudgeUpwards(globeOffset))
        pin.events.globeEnter(e) if pos?

      dragReset: () ->
        pin.well.classList.remove('empty')
        pin.drag.hidden = true

        Util.postForm 'pin',
          csrf_token: Util.csrfToken()

        Pin.transitionMode(pin, 'rest')

      globeEnter: (e) ->
        pin.drag.hidden = true
        pin.globe.interaction.dragPin = Globe.makePin(pin.globe.gl, true)
        pin.globe.gl.scene.add(pin.globe.interaction.dragPin)
        pin.events.globeMove(e)

        Globe.setCaption()

        Globe.transitionMode(pin.globe, 'hoveringWithPin')
        Pin.transitionMode(pin, 'dragGlobe')

      globeMove: (e) ->
        globeOffset = Pin.globeOffset(pin.globe.container, e)
        pos = Globe.raycast(pin.globe.gl, Pin.nudgeUpwards(globeOffset))
        Globe.positionPin(pin.globe.gl, pin.globe.interaction.dragPin, pos) if pos?
        pin.events.globeLeave(e) if !pos?

      globeLeave: (e) ->
        pin.globe.gl.scene.remove(pin.globe.interaction.dragPin)
        pin.globe.interaction.dragPin = null

        pin.events.dragMove(e)
        pin.drag.hidden = false

        Globe.transitionMode(pin.globe, 'rest')
        Pin.transitionMode(pin, 'dragVoid')

      globeUp: (e) ->
        gl = pin.globe.gl
        gl.scene.remove(pin.globe.interaction.dragPin)
        pin.globe.interaction.dragPin = null
        pin.drag.hidden = true

        globeOffset = Pin.globeOffset(pin.globe.container, e)
        pos = Globe.raycast(gl, Pin.nudgeUpwards(globeOffset))

        if pos?
          glpin = Globe.makePin(gl, true)
          glpin.fingerprint = Util.myFingerprint()
          Globe.positionPin(gl, glpin, pos)
          gl.pins.fingerprints[glpin.fingerprint] = glpin
          gl.pins.add(glpin)
          latLon = Globe.vectorToLatLon(pos)

          Util.postForm 'pin',
            csrf_token: Util.csrfToken()
            lat: latLon.lat
            lon: latLon.lon

        Globe.transitionMode(pin.globe, 'rest')
        Pin.transitionMode(pin, 'rest')

    Pin.transitionMode(pin, 'rest')


  transitionMode: (pin, mode) ->
    targets = {well: pin.well, document: document}

    if pin.mode?
      for binding in Pin.eventModes[pin.mode]
        targets[binding[0]].removeEventListener(binding[1], pin.events[binding[2]])

    for binding in Pin.eventModes[mode]
      targets[binding[0]].addEventListener(binding[1], pin.events[binding[2]])

    pin.mode = mode

  scalePin: (pin, scale) ->
    pin.style.transform = 'scale(' + scale + ',' + scale + ')'
    pin.getElementsByClassName('outline')[0].style.strokeWidth = 2 / scale

  elementPosition: (el) ->
    new THREE.Vector2(el.offsetLeft, el.offsetTop)

  mouse: (e) ->
    new THREE.Vector2(e.clientX, e.clientY)

  wellOffset: (well, e) ->
    Pin.mouse(e).sub(Pin.elementPosition(well.parentNode))

  globeOffset: (globeContainer, e) ->
    Pin.mouse(e).sub(Pin.elementPosition(globeContainer))

  nudgeUpwards: (pos) ->
    pos.clone().add(new THREE.Vector2(0, -8))

  clamp: (limits, x) ->
    limits = [limits[1], limits[0]] if limits[1] < limits[0]
    if x < limits[0]
      limits[0]
    else if x > limits[1]
      limits[1]
    else
      x
  
  interpolate: (domain, range, easing, x) ->
    x = Pin.clamp(domain, x)
    s = (x-domain[0]) / (domain[1]-domain[0])
    s = easing(s) if easing?
    s * (range[1]-range[0]) + range[0]

if Detector.webgl
  spinner = Util.id('spinner-container')
  spinner.style.display = null
  Nav.init()
  Globe.loadEverything (textures, xhr) ->
    spinner.parentNode.removeChild(spinner)
    globe = Globe.init(Util.id('gl'), textures, xhr)
    globe.container.style.display = null
    pin = Pin.init(globe)
else
  container = Util.id('gl')
  container.style.display = null
  Util.element 'img',
      src: 'images/nogl.jpg'
      alt: "WebGL is missing"
    .inject(container)
