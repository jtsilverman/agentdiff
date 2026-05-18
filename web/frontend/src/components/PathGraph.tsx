'use client';

import { useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  Handle,
  Position,
  type Node as RFNode,
  type Edge as RFEdge,
  type NodeProps,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import type {
  PathGraph as PathGraphData,
  OverlayResult,
  GraphEdge,
} from '@/lib/types';

export interface PathGraphProps {
  graph: PathGraphData;
  overlay?: OverlayResult;
}

type OverlayState = 'matched' | 'divergent' | 'unvisited';

interface OutgoingBranch {
  to: string;
  weight: number;
}

interface ToolNodeData extends Record<string, unknown> {
  label: string;
  overlayState: OverlayState;
  outgoing: OutgoingBranch[];
}

function ToolNode({ data }: NodeProps) {
  const d = data as ToolNodeData;
  const colorClass =
    d.overlayState === 'divergent'
      ? 'border-yellow-500 bg-yellow-900/30 text-yellow-100'
      : d.overlayState === 'matched'
        ? 'border-green-500 bg-green-900/30 text-green-100'
        : 'border-gray-600 bg-gray-800 text-gray-100';
  const isBranch = d.outgoing.length >= 2;
  return (
    <div
      data-overlay-state={d.overlayState}
      className={`rounded border px-3 py-2 text-sm font-medium ${colorClass}`}
    >
      <Handle type="target" position={Position.Left} />
      <div>{d.label}</div>
      {isBranch && (
        <ul className="mt-1 space-y-0.5 text-xs font-normal opacity-80">
          {d.outgoing.map((o) => (
            <li key={o.to}>
              {'→ '}
              {o.to}
              {' '}
              <span className="font-mono">{`${Math.round(o.weight * 100)}%`}</span>
            </li>
          ))}
        </ul>
      )}
      <Handle type="source" position={Position.Right} />
    </div>
  );
}

const nodeTypes = { tool: ToolNode };

function classifyNode(id: string, overlay?: OverlayResult): OverlayState {
  if (!overlay) return 'unvisited';
  if (overlay.divergence_points.some((d) => d.node_id === id)) return 'divergent';
  if (overlay.matched_node_ids.includes(id)) return 'matched';
  return 'unvisited';
}

function classifyEdge(
  edge: GraphEdge,
  overlay?: OverlayResult,
): { state: OverlayState; color: string } {
  if (!overlay) return { state: 'unvisited', color: '#6b7280' };
  const id = `${edge.from}->${edge.to}`;
  if (overlay.matched_edge_ids.includes(id)) {
    return { state: 'matched', color: '#22c55e' };
  }
  const isDivergent = overlay.divergence_points.some(
    (d) => d.node_id === edge.from && d.this_trace_chose === edge.to,
  );
  if (isDivergent) return { state: 'divergent', color: '#eab308' };
  return { state: 'unvisited', color: '#6b7280' };
}

const NODE_WIDTH = 180;
const NODE_HEIGHT = 60;

function computePositions(graph: PathGraphData): Record<string, { x: number; y: number }> {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: 'LR', nodesep: 40, ranksep: 100 });
  g.setDefaultEdgeLabel(() => ({}));
  for (const n of graph.nodes) g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  for (const e of graph.edges) g.setEdge(e.from, e.to);
  dagre.layout(g);
  const positions: Record<string, { x: number; y: number }> = {};
  for (const id of g.nodes()) {
    const { x, y } = g.node(id);
    positions[id] = { x: x - NODE_WIDTH / 2, y: y - NODE_HEIGHT / 2 };
  }
  return positions;
}

export default function PathGraph({ graph, overlay }: PathGraphProps) {
  const { nodes, edges } = useMemo(() => {
    if (graph.nodes.length === 0) {
      return { nodes: [] as RFNode<ToolNodeData>[], edges: [] as RFEdge[] };
    }

    const positions = computePositions(graph);

    const outgoingByFrom = new Map<string, OutgoingBranch[]>();
    for (const e of graph.edges) {
      const list = outgoingByFrom.get(e.from) ?? [];
      list.push({ to: e.to, weight: e.weight });
      outgoingByFrom.set(e.from, list);
    }
    Array.from(outgoingByFrom.values()).forEach((list) => {
      list.sort((a, b) => b.weight - a.weight);
    });

    const rfNodes: RFNode<ToolNodeData>[] = graph.nodes.map((n) => ({
      id: n.id,
      type: 'tool',
      position: positions[n.id] ?? { x: 0, y: 0 },
      data: {
        label: n.tool_name,
        overlayState: classifyNode(n.id, overlay),
        outgoing: outgoingByFrom.get(n.id) ?? [],
      },
    }));

    const rfEdges: RFEdge[] = graph.edges.map((e) => {
      const { state, color } = classifyEdge(e, overlay);
      return {
        id: `${e.from}->${e.to}`,
        source: e.from,
        target: e.to,
        data: { state },
        animated: state === 'matched',
        style: { stroke: color, strokeWidth: state === 'divergent' ? 3 : 2 },
      };
    });

    return { nodes: rfNodes, edges: rfEdges };
  }, [graph, overlay]);

  if (graph.nodes.length === 0) {
    return (
      <div className="rounded border border-gray-700 p-6 text-sm text-gray-400">
        No graph data yet.
      </div>
    );
  }

  return (
    <div className="h-[500px] w-full rounded border border-gray-700" style={{ height: 500 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}
