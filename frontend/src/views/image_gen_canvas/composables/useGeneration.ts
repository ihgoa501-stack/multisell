import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { imageGenCanvasApi } from '@/api/modules/imageGenCanvas'

export function useGeneration() {
  const message = useMessage()
  const generating = ref(false)
  const videoGenerating = ref(false)
  const videoJobId = ref<string | null>(null)
  const videoUrl = ref<string | null>(null)

  async function inpaint(
    imageUrl: string,
    maskBase64: string,
    prompt: string,
    negativePrompt?: string
  ) {
    generating.value = true
    try {
      const resp = await imageGenCanvasApi.inpaint({
        image_url: imageUrl,
        mask_base64: maskBase64,
        prompt,
        negative_prompt: negativePrompt,
      })
      const data = resp.data as any
      message.success('重绘完成')
      return data?.image_url || null
    } catch (e: any) {
      message.error(e?.response?.data?.message || '重绘失败')
      return null
    } finally {
      generating.value = false
    }
  }

  async function outpaint(
    imageUrl: string,
    direction: string,
    prompt: string,
    expandRatio?: number
  ) {
    generating.value = true
    try {
      const resp = await imageGenCanvasApi.outpaint({
        image_url: imageUrl,
        direction,
        prompt,
        expand_ratio: expandRatio,
      })
      const data = resp.data as any
      message.success('扩图完成')
      return data?.image_url || null
    } catch (e: any) {
      message.error(e?.response?.data?.message || '扩图失败')
      return null
    } finally {
      generating.value = false
    }
  }

  async function startVideoGen(prompt: string, imageUrl?: string) {
    videoGenerating.value = true
    videoUrl.value = null
    try {
      const resp = await imageGenCanvasApi.generateVideo({ prompt, image_url: imageUrl })
      const data = resp.data as any
      videoJobId.value = data.job_id
      pollVideoStatus(data.job_id)
    } catch (e: any) {
      message.error(e?.response?.data?.message || '视频生成失败')
      videoGenerating.value = false
    }
  }

  async function pollVideoStatus(jobId: string) {
    const interval = setInterval(async () => {
      try {
        const resp = await imageGenCanvasApi.getVideoStatus(jobId)
        const data = resp.data as any
        if (data.status === 'done') {
          videoUrl.value = data.video_url
          videoGenerating.value = false
          videoJobId.value = null
          message.success('视频生成完成')
          clearInterval(interval)
        } else if (data.status === 'failed') {
          message.error(data.error || '视频生成失败')
          videoGenerating.value = false
          videoJobId.value = null
          clearInterval(interval)
        }
      } catch {
        clearInterval(interval)
        videoGenerating.value = false
      }
    }, 3000)
  }

  async function createSlideshow(imageUrls: string[], durationPerFrame?: number) {
    videoGenerating.value = true
    try {
      const resp = await imageGenCanvasApi.createSlideshow({
        image_urls: imageUrls,
        duration_per_frame: durationPerFrame,
      })
      const data = resp.data as any
      videoUrl.value = data.video_url
      message.success('视频合成完成')
      return data.video_url
    } catch (e: any) {
      message.error(e?.response?.data?.message || '视频合成失败')
      return null
    } finally {
      videoGenerating.value = false
    }
  }

  return {
    generating,
    videoGenerating,
    videoJobId,
    videoUrl,
    inpaint,
    outpaint,
    startVideoGen,
    createSlideshow,
  }
}
