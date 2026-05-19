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
import type { CounterfactualComparison } from '@/lib/types';

export interface CounterfactualGraphProps {
  comparison: CounterfactualComparison;
}

type BranchState = 'shared' | 'original' | 'counterfactual' | 'divergence';

interface CounterfactualNodeData extends Record<string, unknown> {
  label: string;
  state: BranchState;
}

function CounterfactualNode({ data }: NodeProps) {
  const d = data as CounterfactualNodeData;
  const colorClass =
    d.state === 'shared'
      ? 'border-green-500 bg-green-900/30 text-green-100'
      : d.state === 'divergence'
        ? 'border-yellow-500 bg-yellow-900/40 text-yellow-100'
        : d.state === 'original'
          ? 'border-blue-500 bg-blue-900/30 text-blue-100'
          : 'border-purple-500 bg-purple-900/30 text-purple-100';
  return (
    <div
      data-overlay-state={d.state}
      className={`rounded border px-3 py-2 text-sm font-medium ${colorClass}`}
    >
      <Handle type="target" position={Position.Left} />
      <div>{d.label}</div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}

const nodeTypes = { cf: CounterfactualNode };

const NODE_WIDTH = 180;
const NODE_HEIGHT = 60;

interface BuiltNode {
  id: string;
  label: string;
  state: BranchState;
}

interface BuiltEdge {
  from: string;
  to: string;
  branch: BranchState;
}

function buildGraph(comparison: CounterfactualComparison): {
  nodes: BuiltNode[];
  edges: BuiltEdge[];
} {
  const { original_path, new_path, divergence_step } = comparison;
  const fork = Math.max(0, Math.min(divergence_step, original_path.length, new_path.length));

  const nodes: BuiltNode[] = [];
  const edges: BuiltEdge[] = [];

  // Shared prefix: indices 0..fork-1.
  let prevSharedId: string | null = null;
  for (let i = 0; i < fork; i++) {
    const id = `shared-${i}`;
    nodes.push({ id, label: original_path[i], state: 'shared' });
    if (prevSharedId) edges.push({ from: prevSharedId, to: id, branch: 'shared' });
    prevSharedId = id;
  }

  // Original branch starting at fork.
  let prevOrigId: string | null = null;
  for (let i = fork; i < original_path.length; i++) {
    const id = `orig-${i}`;
    const state: BranchState = i === fork ? 'divergence' : 'original';
    nodes.push({ id, label: original_path[i], state });
    if (i === fork && prevSharedId) {
      edges.push({ from: prevSharedId, to: id, branch: 'original' });
    } else if (prevOrigId) {
      edges.push({ from: prevOrigId, to: id, branch: 'original' });
    }
    prevOrigId = id;
  }

  // Counterfactual branch starting at fork.
  let prevCfId: string | null = null;
  for (let i = fork; i < new_path.length; i++) {
    const id = `cf-${i}`;
    const state: BranchState = i === fork ? 'divergence' : 'counterfactual';
    nodes.push({ id, label: new_path[i], state });
    if (i === fork && prevSharedId) {
      edges.push({ from: prevSharedId, to: id, branch: 'counterfactual' });
    } else if (prevCfId) {
      edges.push({ from: prevCfId, to: id, branch: 'counterfactual' });
    }
    prevCfId = id;
  }

  return { nodes, edges };
}

function computePositions(
  builtNodes: BuiltNode[],
  builtEdges: BuiltEdge[],
): Record<string, { x: number; y: number }> {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: 'LR', nodesep: 60, ranksep: 120 });
  g.setDefaultEdgeLabel(() => ({}));
  for (const n of builtNodes) g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  for (const e of builtEdges) g.setEdge(e.from, e.to);
  dagre.layout(g);
  const positions: Record<string, { x: number; y: number }> = {};
  for (const id of g.nodes()) {
    const { x, y } = g.node(id);
    positions[id] = { x: x - NODE_WIDTH / 2, y: y - NODE_HEIGHT / 2 };
  }
  return positions;
}

function edgeColor(branch: BranchState): string {
  switch (branch) {
    case 'shared':
      return '#22c55e';
    case 'original':
      return '#3b82f6';
    case 'counterfactual':
      return '#a855f7';
    default:
      return '#6b7280';
  }
}

export default function CounterfactualGraph({ comparison }: CounterfactualGraphProps) {
  const { nodes, edges } = useMemo(() => {
    const built = buildGraph(comparison);
    if (built.nodes.length === 0) {
      return { nodes: [] as RFNode<CounterfactualNodeData>[], edges: [] as RFEdge[] };
    }
    const positions = computePositions(built.nodes, built.edges);
    const rfNodes: RFNode<CounterfactualNodeData>[] = built.nodes.map((n) => ({
      id: n.id,
      type: 'cf',
      position: positions[n.id] ?? { x: 0, y: 0 },
      data: { label: n.label, state: n.state },
    }));
    const rfEdges: RFEdge[] = built.edges.map((e) => ({
      id: `${e.from}->${e.to}`,
      source: e.from,
      target: e.to,
      style: { stroke: edgeColor(e.branch), strokeWidth: 2 },
    }));
    return { nodes: rfNodes, edges: rfEdges };
  }, [comparison]);

  if (nodes.length === 0) {
    return (
      <div className="rounded border border-gray-700 p-6 text-sm text-gray-400">
        No counterfactual data.
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
