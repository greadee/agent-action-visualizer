import { OrbitControls } from '@react-three/drei'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useRef } from 'react'
import { Object3D, Quaternion, Vector3 } from 'three'
import type { OrbitControls as OrbitControlsImpl } from 'three-stdlib'
import type { Vec3 } from '../graph/types'
import {
  easeInOutCubic,
  focusCameraPosition,
  focusDurationMs,
} from './focusMath'

interface Transition {
  elapsedMs: number
  durationMs: number
  fromPosition: Vector3
  toPosition: Vector3
  fromQuaternion: Quaternion
  toQuaternion: Quaternion
  fromTarget: Vector3
  toTarget: Vector3
}

export function CameraFocusController({
  focus,
  recenterKey,
}: {
  focus?: Vec3
  recenterKey: number
}) {
  const controls = useRef<OrbitControlsImpl>(null)
  const transition = useRef<Transition | undefined>(undefined)
  const camera = useThree((state) => state.camera)

  useEffect(() => {
    if (!focus || !controls.current) return
    const toTarget = new Vector3(...focus)
    const distance = controls.current.target.distanceTo(toTarget)
    const currentOrbitDistance = camera.position.distanceTo(
      controls.current.target,
    )
    const isOverview = toTarget.lengthSq() < 0.0001
    const orbitDistance = isOverview
      ? 28
      : Math.min(11, Math.max(8, currentOrbitDistance))
    const toPosition = focusCameraPosition(
      focus,
      [camera.position.x, camera.position.y, camera.position.z],
      orbitDistance,
    )
    const lookAt = new Object3D()
    lookAt.position.copy(toPosition)
    lookAt.lookAt(toTarget)
    const reducedMotion = window.matchMedia(
      '(prefers-reduced-motion: reduce)',
    ).matches
    const next: Transition = {
      elapsedMs: 0,
      durationMs: reducedMotion ? 0 : focusDurationMs(distance),
      fromPosition: camera.position.clone(),
      toPosition,
      fromQuaternion: camera.quaternion.clone(),
      toQuaternion: lookAt.quaternion.clone(),
      fromTarget: controls.current.target.clone(),
      toTarget,
    }
    transition.current = next
    if (reducedMotion) {
      camera.position.copy(next.toPosition)
      camera.quaternion.copy(next.toQuaternion)
      controls.current.target.copy(next.toTarget)
      controls.current.update()
      transition.current = undefined
    }
  }, [camera, focus, recenterKey])

  useFrame((_, delta) => {
    const active = transition.current
    if (!active || !controls.current) return
    active.elapsedMs += delta * 1000
    const progress = easeInOutCubic(active.elapsedMs / active.durationMs)
    camera.position.lerpVectors(
      active.fromPosition,
      active.toPosition,
      progress,
    )
    camera.quaternion.slerpQuaternions(
      active.fromQuaternion,
      active.toQuaternion,
      progress,
    )
    controls.current.target.lerpVectors(
      active.fromTarget,
      active.toTarget,
      progress,
    )
    controls.current.update()
    if (progress >= 1) transition.current = undefined
  })

  return (
    <OrbitControls
      ref={controls}
      makeDefault
      enableDamping
      dampingFactor={0.08}
      onStart={() => {
        transition.current = undefined
      }}
    />
  )
}
