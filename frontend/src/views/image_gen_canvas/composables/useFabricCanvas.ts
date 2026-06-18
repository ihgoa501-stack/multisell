import { ref, onMounted, onUnmounted, type Ref } from 'vue'
import * as fabric from 'fabric'

export function useFabricCanvas(canvasEl: Ref<HTMLCanvasElement | null>) {
  const canvas = ref<fabric.Canvas | null>(null)
  const zoom = ref(1)
  const selectedObject = ref<fabric.Object | null>(null)

  function initCanvas() {
    if (!canvasEl.value) return
    const el = canvasEl.value
    const parent = el.parentElement!
    const c = new fabric.Canvas(el, {
      width: parent.clientWidth,
      height: parent.clientHeight,
      selection: true,
      preserveObjectStacking: true,
      backgroundColor: '#f0f0f0',
    })

    c.on('mouse:wheel', (opt) => {
      const delta = opt.e.deltaY
      let z = c.getZoom()
      z *= 0.999 ** delta
      z = Math.max(0.1, Math.min(5, z))
      c.zoomToPoint({ x: (opt.e as MouseEvent).offsetX, y: (opt.e as MouseEvent).offsetY }, z)
      zoom.value = z
      opt.e.preventDefault()
      opt.e.stopPropagation()
    })

    c.on('selection:created', (e) => { selectedObject.value = e.selected?.[0] || null })
    c.on('selection:updated', (e) => { selectedObject.value = e.selected?.[0] || null })
    c.on('selection:cleared', () => { selectedObject.value = null })

    canvas.value = c
  }

  function addImage(url: string) {
    if (!canvas.value) return
    fabric.Image.fromURL(url, (img) => {
      img.set({
        left: 100 + Math.random() * 200,
        top: 100 + Math.random() * 200,
        cornerColor: '#1890ff',
        cornerSize: 8,
        transparentCorners: false,
      })
      canvas.value!.add(img)
      canvas.value!.setActiveObject(img)
      canvas.value!.renderAll()
    }, { crossOrigin: 'anonymous' })
  }

  function addText(text: string) {
    if (!canvas.value) return
    const tb = new fabric.Textbox(text, {
      left: 200,
      top: 200,
      fontSize: 32,
      fill: '#333',
      fontFamily: 'Arial',
    })
    canvas.value.add(tb)
    canvas.value.setActiveObject(tb)
    canvas.value.renderAll()
  }

  function deleteSelected() {
    if (!canvas.value || !selectedObject.value) return
    canvas.value.remove(selectedObject.value)
    canvas.value.renderAll()
    selectedObject.value = null
  }

  function bringForward() {
    if (!canvas.value || !selectedObject.value) return
    canvas.value.bringForward(selectedObject.value)
    canvas.value.renderAll()
  }

  function sendBackward() {
    if (!canvas.value || !selectedObject.value) return
    canvas.value.sendBackwards(selectedObject.value)
    canvas.value.renderAll()
  }

  function exportImage(format: 'png' | 'jpeg' = 'png'): string {
    if (!canvas.value) return ''
    return canvas.value.toDataURL({ format, multiplier: 2 })
  }

  function getCanvasJSON(): object {
    if (!canvas.value) return {}
    return canvas.value.toJSON()
  }

  function loadFromJSON(json: object) {
    if (!canvas.value) return
    canvas.value.loadFromJSON(json, () => {
      canvas.value!.renderAll()
    })
  }

  function setZoom(z: number) {
    if (!canvas.value) return
    canvas.value.setZoom(z)
    zoom.value = z
    canvas.value.renderAll()
  }

  function fitToScreen() {
    if (!canvas.value) return
    canvas.value.setZoom(1)
    zoom.value = 1
    canvas.value.renderAll()
  }

  onMounted(() => {
    initCanvas()
  })

  onUnmounted(() => {
    canvas.value?.dispose()
  })

  return {
    canvas,
    zoom,
    selectedObject,
    addImage,
    addText,
    deleteSelected,
    bringForward,
    sendBackward,
    exportImage,
    getCanvasJSON,
    loadFromJSON,
    setZoom,
    fitToScreen,
  }
}
