'use client';

import { useState } from 'react';
import { Button, Card, Empty, Input, InputNumber, message, Modal, Space, Table, Tag, Typography } from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  ArrowLeftOutlined,
  SaveOutlined,
  FontSizeOutlined,
  PictureOutlined,
  BgColorsOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import ConfirmDialog from '@/components/ui/ConfirmDialog';
import PageContainer from '@/components/ui/PageContainer';

/* ------------- types ------------- */

interface TextLayer {
  id: string;
  type: 'text';
  text: string;
  fontSize: number;
  color: string;
  x: number;
  y: number;
}

interface ImageLayer {
  id: string;
  type: 'image';
  src: string;
  width: number;
  height: number;
  x: number;
  y: number;
}

type Layer = TextLayer | ImageLayer;

interface CanvasData {
  width: number;
  height: number;
  background: string;
  items: Layer[];
}

interface CanvasRecord {
  id: number;
  product_id: number;
  name: string;
  layers: CanvasData | null;
  thumbnail: string;
  created_at: string;
  updated_at: string;
}

const CANVAS_W = 800;
const CANVAS_H = 600;
const SCALE = 400 / CANVAS_W;

function emptyCanvas(): CanvasData {
  return { width: CANVAS_W, height: CANVAS_H, background: '#FFFFFF', items: [] };
}

function genId(): string {
  return Math.random().toString(36).slice(2, 10);
}

function fmtDate(v: string): string {
  return v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-';
}

/* ------------- layer edit modal ------------- */

function LayerEditModal({
  open,
  layer,
  onOk,
  onCancel,
}: {
  open: boolean;
  layer: Layer | null;
  onOk: (l: Layer) => void;
  onCancel: () => void;
}) {
  const [text, setText] = useState(layer?.type === 'text' ? layer.text : '');
  const [fontSize, setFontSize] = useState(layer?.type === 'text' ? layer.fontSize : 24);
  const [color, setColor] = useState(layer?.type === 'text' ? layer.color : '#000000');
  const [src, setSrc] = useState(layer?.type === 'image' ? layer.src : '');
  const [imgW, setImgW] = useState(layer?.type === 'image' ? layer.width : 200);
  const [imgH, setImgH] = useState(layer?.type === 'image' ? layer.height : 200);
  const [x, setX] = useState(layer?.x ?? 0);
  const [y, setY] = useState(layer?.y ?? 0);

  const handleOk = () => {
    if (!layer) return;
    if (layer.type === 'text') {
      if (!text.trim()) { message.warning('请输入文本'); return; }
      onOk({ ...layer, text, fontSize, color, x, y });
    } else {
      if (!src.trim()) { message.warning('请输入图片 URL'); return; }
      onOk({ ...layer, src, width: imgW, height: imgH, x, y });
    }
  };

  return (
    <Modal title={layer?.type === 'text' ? '编辑文本' : '编辑图片'} open={open} onOk={handleOk} onCancel={onCancel} width={440} destroyOnClose>
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        <div style={{ display: 'flex', gap: 12 }}>
          <div><Typography.Text type="secondary">X</Typography.Text><InputNumber value={x} onChange={(v) => setX(v ?? 0)} style={{ width: 80 }} /></div>
          <div><Typography.Text type="secondary">Y</Typography.Text><InputNumber value={y} onChange={(v) => setY(v ?? 0)} style={{ width: 80 }} /></div>
        </div>
        {layer?.type === 'text' ? (
          <>
            <div>
              <Typography.Text type="secondary">文本</Typography.Text>
              <Input.TextArea rows={2} value={text} onChange={(e) => setText(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: 12 }}>
              <div><Typography.Text type="secondary">字号</Typography.Text><InputNumber value={fontSize} onChange={(v) => setFontSize(v ?? 24)} min={8} max={200} /></div>
              <div>
                <Typography.Text type="secondary">颜色</Typography.Text>
                <div><input type="color" value={color} onChange={(e) => setColor(e.target.value)} style={{ padding: 0, border: '1px solid #d9d9d9', borderRadius: 4, cursor: 'pointer', width: 60, height: 32 }} /></div>
              </div>
            </div>
          </>
        ) : (
          <>
            <div>
              <Typography.Text type="secondary">图片 URL</Typography.Text>
              <Input value={src} onChange={(e) => setSrc(e.target.value)} placeholder="https://..." />
            </div>
            <div style={{ display: 'flex', gap: 12 }}>
              <div><Typography.Text type="secondary">宽</Typography.Text><InputNumber value={imgW} onChange={(v) => setImgW(v ?? 200)} min={1} /></div>
              <div><Typography.Text type="secondary">高</Typography.Text><InputNumber value={imgH} onChange={(v) => setImgH(v ?? 200)} min={1} /></div>
            </div>
          </>
        )}
      </Space>
    </Modal>
  );
}

/* ------------- canvas preview ------------- */

function CanvasPreview({ data }: { data: CanvasData }) {
  const w = data.width * SCALE;
  const h = data.height * SCALE;
  return (
    <div style={{ width: w, height: h, background: data.background, position: 'relative', border: '1px solid #d9d9d9', borderRadius: 4, overflow: 'hidden', flexShrink: 0 }}>
      {data.items.map((layer) => {
        const sx = layer.x * SCALE;
        const sy = layer.y * SCALE;
        if (layer.type === 'text') {
          return (
            <span key={layer.id} style={{ position: 'absolute', left: sx, top: sy, fontSize: layer.fontSize * SCALE, color: layer.color, whiteSpace: 'pre-wrap', userSelect: 'none', pointerEvents: 'none' }}>
              {layer.text}
            </span>
          );
        }
        return (
          <img key={layer.id} src={layer.src} alt="" style={{ position: 'absolute', left: sx, top: sy, width: layer.width * SCALE, height: layer.height * SCALE, userSelect: 'none', pointerEvents: 'none' }} />
        );
      })}
    </div>
  );
}

/* ------------- main page ------------- */

export default function CanvasPage() {
  const qc = useQueryClient();
  const [mode, setMode] = useState<'list' | 'edit'>('list');
  const [canvas, setCanvas] = useState<CanvasRecord | null>(null);
  const [canvasData, setCanvasData] = useState<CanvasData>(emptyCanvas);
  const [canvasName, setCanvasName] = useState('');
  const [productId, setProductId] = useState(1);
  const [page, setPage] = useState(1);
  const [deleteTarget, setDeleteTarget] = useState<CanvasRecord | null>(null);
  const [layerModal, setLayerModal] = useState<{ layer: Layer; index: number } | null>(null);
  const [deleteCanvasId, setDeleteCanvasId] = useState<number | null>(null);

  const { data: listData, isLoading } = useQuery({
    queryKey: ['canvases', page],
    queryFn: async () => {
      const res = await apiClient.getPage<CanvasRecord>('/v1/imagegen/canvas', { page: String(page), size: '10' });
      return { items: res.data ?? [], total: res.total ?? 0 };
    },
    enabled: mode === 'list',
  });

  const create = useMutation({
    mutationFn: (body: Record<string, unknown>) => apiClient.post('/v1/imagegen/canvas', body),
    onSuccess: () => { message.success('已创建'); qc.invalidateQueries({ queryKey: ['canvases'] }); },
    onError: (e: Error) => message.error('创建失败: ' + e.message),
  });
  const updateRec = useMutation({
    mutationFn: ({ id, body }: { id: number; body: Record<string, unknown> }) => apiClient.put(`/v1/imagegen/canvas/${id}`, body),
    onSuccess: () => { message.success('已保存'); qc.invalidateQueries({ queryKey: ['canvases'] }); },
    onError: (e: Error) => message.error('保存失败: ' + e.message),
  });
  const del = useMutation({
    mutationFn: (id: number) => apiClient.delete(`/v1/imagegen/canvas/${id}`),
    onSuccess: () => { message.success('已删除'); setDeleteTarget(null); qc.invalidateQueries({ queryKey: ['canvases'] }); },
    onError: (e: Error) => message.error('删除失败: ' + e.message),
  });

  const handleEdit = (c: CanvasRecord) => {
    setCanvas(c);
    setCanvasName(c.name);
    setProductId(c.product_id);
    setCanvasData(c.layers ?? emptyCanvas());
    setMode('edit');
  };

  const handleNew = () => {
    setCanvas(null);
    setCanvasName('未命名画布');
    setProductId(1);
    setCanvasData(emptyCanvas());
    setMode('edit');
  };

  const handleSave = async () => {
    if (!canvasName.trim()) { message.warning('请输入画布名称'); return; }
    const body = { product_id: productId, name: canvasName, layers: JSON.stringify(canvasData), thumbnail: '' };
    if (canvas) {
      await updateRec.mutateAsync({ id: canvas.id, body });
    } else {
      await create.mutateAsync(body);
    }
    setMode('list');
  };

  const handleBack = () => {
    setMode('list');
    setDeleteCanvasId(null);
  };

  const addTextLayer = () => {
    const l: TextLayer = { id: genId(), type: 'text', text: '双击编辑文本', fontSize: 24, color: '#000000', x: 50, y: 50 };
    setCanvasData((d) => ({ ...d, items: [...d.items, l] }));
  };

  const addImageLayer = () => {
    const l: ImageLayer = { id: genId(), type: 'image', src: 'https://via.placeholder.com/200', width: 200, height: 200, x: 50, y: 50 };
    setCanvasData((d) => ({ ...d, items: [...d.items, l] }));
  };

  const deleteLayer = (id: string) => {
    setCanvasData((d) => ({ ...d, items: d.items.filter((l) => l.id !== id) }));
  };

  const openLayerEdit = (idx: number) => {
    setLayerModal({ layer: canvasData.items[idx], index: idx });
  };

  const saveLayer = (l: Layer) => {
    if (layerModal == null) return;
    setCanvasData((d) => {
      const items = [...d.items];
      items[layerModal.index] = l;
      return { ...d, items };
    });
    setLayerModal(null);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    { title: '图层数', width: 70, render: (_: unknown, r: CanvasRecord) => (r.layers?.items?.length ?? 0) },
    {
      title: '背景', width: 100,
      render: (_: unknown, r: CanvasRecord) =>
        r.layers?.background ? <Tag color={r.layers.background}>{r.layers.background}</Tag> : '-',
    },
    { title: '更新于', dataIndex: 'updated_at', width: 160, render: (v: string) => fmtDate(v) },
    {
      title: '操作', key: 'actions', width: 140,
      render: (_: unknown, r: CanvasRecord) => (
        <Space size="small">
          <Button size="small" type="link" icon={<EditOutlined />} onClick={() => handleEdit(r)}>编辑</Button>
          <Button size="small" type="link" danger icon={<DeleteOutlined />} onClick={() => setDeleteTarget(r)}>删除</Button>
        </Space>
      ),
    },
  ];

  // List mode
  if (mode === 'list') {
    return (
      <PageContainer title="AI 画布编辑器" extra={<Button type="primary" icon={<PlusOutlined />} onClick={handleNew}>新建画布</Button>}>
        <Table
          rowKey="id"
          loading={isLoading}
          dataSource={listData?.items}
          columns={columns}
          pagination={{ current: page, pageSize: 10, total: listData?.total ?? 0, onChange: (p) => setPage(p) }}
        />
        <ConfirmDialog
          open={!!deleteTarget} title="删除画布" content="确定要删除此画布吗？此操作不可撤销。"
          okType="danger" okText="确认删除" confirmLoading={del.isPending} risk="high"
          onOk={() => deleteTarget && del.mutate(deleteTarget.id)}
          onCancel={() => setDeleteTarget(null)}
        />
      </PageContainer>
    );
  }

  // Edit mode
  return (
    <PageContainer
      title={canvas ? `编辑画布: ${canvas.name}` : '新建画布'}
      extra={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>返回</Button>
          {canvas && <Button danger icon={<DeleteOutlined />} onClick={() => setDeleteCanvasId(canvas.id)}>删除</Button>}
          <Button type="primary" icon={<SaveOutlined />} loading={create.isPending || updateRec.isPending} onClick={handleSave}>保存</Button>
        </Space>
      }
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center' }}>
          <Input placeholder="画布名称" value={canvasName} onChange={(e) => setCanvasName(e.target.value)} style={{ width: 240 }} />
          <InputNumber placeholder="产品 ID" value={productId} onChange={(v) => setProductId(v ?? 1)} min={1} style={{ width: 120 }} />
          <label style={{ display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>
            <BgColorsOutlined />
            <Typography.Text type="secondary">背景</Typography.Text>
            <input
              type="color" value={canvasData.background}
              onChange={(e) => setCanvasData((d) => ({ ...d, background: e.target.value }))}
              style={{ width: 40, height: 28, padding: 0, border: '1px solid #d9d9d9', borderRadius: 4, cursor: 'pointer' }}
            />
          </label>
        </div>

        <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
          <CanvasPreview data={canvasData} />
          <div style={{ flex: 1, minWidth: 280 }}>
            <div style={{ marginBottom: 8, display: 'flex', gap: 8 }}>
              <Button size="small" icon={<FontSizeOutlined />} onClick={addTextLayer}>添加文本</Button>
              <Button size="small" icon={<PictureOutlined />} onClick={addImageLayer}>添加图片</Button>
            </div>
            {canvasData.items.length === 0 ? (
              <Empty description="暂无图层，点击上方按钮添加" style={{ margin: '12px 0' }} />
            ) : (
              <Space direction="vertical" style={{ width: '100%' }} size="small">
                {canvasData.items.map((layer, idx) => (
                  <Card key={layer.id} size="small" style={{ width: '100%' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Space>
                        <Tag>{layer.type === 'text' ? '文本' : '图片'}</Tag>
                        <Typography.Text ellipsis style={{ maxWidth: 180, fontSize: 13 }}>
                          {layer.type === 'text' ? `"${layer.text}"` : `${layer.width}x${layer.height}`}
                        </Typography.Text>
                      </Space>
                      <Space>
                        <Button size="small" type="link" icon={<EditOutlined />} onClick={() => openLayerEdit(idx)} />
                        <Button size="small" type="link" danger icon={<DeleteOutlined />} onClick={() => deleteLayer(layer.id)} />
                      </Space>
                    </div>
                  </Card>
                ))}
              </Space>
            )}
          </div>
        </div>
      </Space>

      <LayerEditModal open={!!layerModal} layer={layerModal?.layer ?? null} onOk={saveLayer} onCancel={() => setLayerModal(null)} />
      <ConfirmDialog
        open={!!deleteCanvasId} title="删除画布" content="确定要删除此画布吗？"
        okType="danger" okText="确认删除" confirmLoading={del.isPending} risk="high"
        onOk={() => { if (deleteCanvasId) { del.mutate(deleteCanvasId); setMode('list'); } }}
        onCancel={() => setDeleteCanvasId(null)}
      />
    </PageContainer>
  );
}
