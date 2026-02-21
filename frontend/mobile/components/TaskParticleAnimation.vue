<template>
  <div class="particle-container" v-if="show">
    <div 
      v-for="particle in particles" 
      :key="particle.id"
      class="particle"
      :style="{
        left: particle.x + 'px',
        top: particle.y + 'px',
        width: particle.size + 'px',
        height: particle.size + 'px',
        opacity: particle.opacity,
        transform: `translate(${particle.vx * particle.life}px, ${particle.vy * particle.life}px) rotate(${particle.rotation * particle.life}deg)`
      }"
    ></div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, watch, onUnmounted } from 'vue';

export default defineComponent({
  name: 'TaskParticleAnimation',
  props: {
    show: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const particles = ref<any[]>([]);
    let animationFrame: number | null = null;

    const createParticles = (x: number, y: number) => {
      const particleCount = 30;
      const newParticles: any[] = [];
      const colors = ['#22c55e', '#4ade80', '#86efac', '#16a34a'];
      
      for (let i = 0; i < particleCount; i++) {
        const angle = Math.random() * Math.PI * 2;
        const speed = 80 + Math.random() * 120;
        newParticles.push({
          id: i,
          x,
          y,
          vx: Math.cos(angle) * speed,
          vy: Math.sin(angle) * speed,
          size: 6 + Math.random() * 8,
          opacity: 1,
          life: 0,
          maxLife: 0.8 + Math.random() * 0.4,
          rotation: -180 + Math.random() * 360,
          color: colors[Math.floor(Math.random() * colors.length)],
        });
      }
      
      particles.value = newParticles;
      animateParticles();
    };

    const animateParticles = () => {
      let allDead = true;
      
      particles.value.forEach(particle => {
        particle.life += 0.025;
        particle.opacity = 1 - (particle.life / particle.maxLife);
        
        if (particle.life < particle.maxLife) {
          allDead = false;
        }
      });
      
      if (!allDead) {
        animationFrame = requestAnimationFrame(animateParticles);
      }
    };

    watch(() => props.show, (newVal) => {
      if (newVal) {
        createParticles(window.innerWidth / 2, window.innerHeight / 2);
      }
    });

    onUnmounted(() => {
      if (animationFrame) {
        cancelAnimationFrame(animationFrame);
      }
    });

    return {
      particles,
      createParticles,
    };
  },
});
</script>

<style scoped>
.particle-container {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 9999;
}

.particle {
  position: absolute;
  background: var(--color-success);
  border-radius: 50%;
}
</style>
