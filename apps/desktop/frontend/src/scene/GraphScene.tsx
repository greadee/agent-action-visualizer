import { Html } from '@react-three/drei'
import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4, Vector3 } from 'three'
import { CameraFocusController } from '../camera/CameraFocusController'
import { nodeColors } from '../graph/palette'
import type { GraphSnapshot } from '../graph/types'
export function GraphScene({
  graph,
  selectedId,
  recenterKey,
  showLabels,
  showStructure,
  onSelect,
}: {
  graph: GraphSnapshot
  selectedId?: string
  recenterKey: number
  showLabels: boolean
  showStructure: boolean
  onSelect: (id: string) => void
}) {
  const mesh = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<number>()
  const selected = graph.nodes.find((n) => n.id === selectedId)
  useLayoutEffect(() => {
    const matrix = new Matrix4()
    graph.nodes.forEach((node, index) => {
      matrix.makeTranslation(...node.position)
      const scale =
        node.kind === 'root' ? 1.7 : node.kind === 'directory' ? 1.3 : 1
      matrix.scale(new Vector3(scale, scale, scale))
      mesh.current?.setMatrixAt(index, matrix)
      mesh.current?.setColorAt(
        index,
        new Color(node.id === selectedId ? '#ffffff' : nodeColors[node.kind]),
      )
    })
    if (mesh.current) {
      mesh.current.instanceMatrix.needsUpdate = true
      if (mesh.current.instanceColor)
        mesh.current.instanceColor.needsUpdate = true
    }
  }, [graph, selectedId])
  const positions = useMemo(() => {
    const byId = new Map(graph.nodes.map((n) => [n.id, n.position]))
    return new Float32Array(
      graph.edges.flatMap((e) => [
        ...(byId.get(e.source) ?? [0, 0, 0]),
        ...(byId.get(e.target) ?? [0, 0, 0]),
      ]),
    )
  }, [graph])
  const hoverNode = hovered === undefined ? undefined : graph.nodes[hovered]
  return (
    <>
      <ambientLight intensity={0.75} />
      <directionalLight position={[5, 8, 6]} intensity={2.5} color="#b9fff0" />
      {showStructure && (
        <lineSegments>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              args={[positions, 3]}
            />
          </bufferGeometry>
          <lineBasicMaterial color="#294154" transparent opacity={0.72} />
        </lineSegments>
      )}
      <instancedMesh
        ref={mesh}
        args={[undefined, undefined, graph.nodes.length]}
        onPointerMove={(e) => {
          e.stopPropagation()
          setHovered(e.instanceId)
        }}
        onPointerOut={() => setHovered(undefined)}
        onClick={(e) => {
          e.stopPropagation()
          if (e.instanceId !== undefined)
            onSelect(graph.nodes[e.instanceId]?.id ?? '')
        }}
      >
        <icosahedronGeometry args={[0.22, 2]} />
        <meshStandardMaterial roughness={0.35} metalness={0.15} />
      </instancedMesh>
      {showLabels && selected && (
        <Html position={selected.position} center distanceFactor={9}>
          <span className="node-label">{selected.path}</span>
        </Html>
      )}
      {showLabels && hoverNode && hoverNode.id !== selectedId && (
        <Html position={hoverNode.position} center distanceFactor={10}>
          <span className="node-label node-label--hover">{hoverNode.path}</span>
        </Html>
      )}
      <CameraFocusController
        focus={selected?.position}
        recenterKey={recenterKey}
      />
    </>
  )
}
