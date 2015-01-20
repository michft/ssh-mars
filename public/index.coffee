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

    Pin.addEvents(pin.well, pin.drag, pin.globe)

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

  addEvents: (well, drag, globe) ->
    mode = null
    offset = {x: 0, y: 0}

    events =
      wellEnter: () -> well.classList.add('hover')
      wellLeave: () -> well.classList.remove('hover')

      wellDown: (e) ->
        well.style.cursor = 'grabbing'
        offset = Pin.wellOffset(well, e)
        mode = Pin.transitionMode(well, events, mode, 'wellPressed')

      wellUp: () ->
        well.style.cursor = null
        mode = Pin.transitionMode(well, events, mode, 'rest')

      wellPull: (e) ->
        dist = offset.distanceTo(Pin.wellOffset(well, e))
        events.dragStart(e) if dist > 10

      dragStart: (e) ->
        events.wellLeave()
        events.wellUp()
        well.classList.add('empty')
        events.dragMove(e)
        drag.hidden = false
        drag.style.transformOrigin = offset.x + 'px ' + offset.y + 'px'

        fingerprint = Util.myFingerprint()
        pins = globe.gl.pins
        pin = pins.fingerprints[fingerprint]
        if pin?
          pins.fingerprints[fingerprint] = null
          pins.remove(pin)

        mode = Pin.transitionMode(well, events, mode, 'dragVoid')

      dragMove: (e) ->
        drag.style.left = (e.clientX - offset.x) + 'px'
        drag.style.top  = (e.clientY - offset.y) + 'px'

        globeOffset = Pin.globeOffset(globe.container, e)
        dist = globeOffset.length()

        easing = (x) -> x * x
        scale = Pin.interpolate([1.54, 0.87], [1, 0.1], easing, dist)
        Pin.scalePin(drag, scale)

        pos = Globe.raycast(globe.gl, Pin.nudgeUpwards(globeOffset))
        events.globeEnter(e) if pos?

      dragReset: () ->
        well.classList.remove('empty')
        drag.hidden = true

        Util.postForm 'pin',
          csrf_token: Util.csrfToken()

        mode = Pin.transitionMode(well, events, mode, 'rest')

      globeEnter: (e) ->
        drag.hidden = true
        globe.interaction.dragPin = Globe.makePin(globe.gl, true)
        globe.gl.scene.add(globe.interaction.dragPin)
        events.globeMove(e)
        mode = Pin.transitionMode(well, events, mode, 'dragGlobe')

      globeMove: (e) ->
        globeOffset = Pin.globeOffset(globe.container, e)
        pos = Globe.raycast(globe.gl, Pin.nudgeUpwards(globeOffset))
        Globe.positionPin(globe.gl, globe.interaction.dragPin, pos) if pos?
        events.globeLeave(e) if !pos?

      globeLeave: (e) ->
        globe.gl.scene.remove(globe.interaction.dragPin)
        globe.interaction.dragPin = null

        events.dragMove(e)
        drag.hidden = false
        mode = Pin.transitionMode(well, events, mode, 'dragVoid')

      globeUp: (e) ->
        globe.gl.scene.remove(globe.interaction.dragPin)
        globe.interaction.dragPin = null
        drag.hidden = true

        globeOffset = Pin.globeOffset(globe.container, e)
        pos = Globe.raycast(globe.gl, Pin.nudgeUpwards(globeOffset))

        if pos?
          pin = Globe.makePin(globe.gl, true)
          pin.fingerprint = Util.myFingerprint()
          Globe.positionPin(globe.gl, pin, pos)
          globe.gl.pins.fingerprints[pin.fingerprint] = pin
          globe.gl.pins.add(pin)
          latLon = Globe.vectorToLatLon(pos)

          Util.postForm 'pin',
            csrf_token: Util.csrfToken()
            lat: latLon.lat
            lon: latLon.lon

        mode = Pin.transitionMode(well, events, mode, 'rest')

    mode = Pin.transitionMode(well, events, mode, 'rest')


  transitionMode: (well, events, prevMode, mode) ->
    targets = {well: well, document: document}

    if prevMode?
      for binding in Pin.eventModes[prevMode]
        targets[binding[0]].removeEventListener(binding[1], events[binding[2]])

    for binding in Pin.eventModes[mode]
      targets[binding[0]].addEventListener(binding[1], events[binding[2]])

    mode

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
      .multiplyScalar(2/800)
      .addScalar(-1)
      .multiply(new THREE.Vector2(1, -1))

  nudgeUpwards: (pos) ->
    pos.clone().add(new THREE.Vector2(0, 0.02))

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

Nav.init()
Globe.loadEverything (textures, xhr) ->
  spinner = Util.id('spinner-container')
  spinner.parentNode.removeChild(spinner)
  globe = Globe.init(Util.id('gl'), textures, xhr)
  globe.container.style.display = null
  pin = Pin.init(globe)
